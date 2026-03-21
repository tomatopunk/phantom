// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

//! Rust CLI for Phantom: REPL and `discover` subcommands.

use clap::{Parser, Subcommand};
use phantom_client::{DebugEvent, EventType, PhantomClient};
use std::io::{self, BufRead, Write};

#[derive(Parser, Debug)]
#[command(name = "phantom-cli")]
#[command(about = "Phantom eBPF debugger client (Rust)")]
struct Args {
    #[arg(short = 'a', long = "agent", default_value = "127.0.0.1:9090")]
    agent: String,
    #[arg(short = 't', long = "token", default_value = "")]
    token: String,
    #[arg(short = 'x', long = "script")]
    script_path: Option<String>,
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// List tracepoints (agent on Linux + tracefs).
    Tracepoints {
        #[arg(short, long, default_value = "")]
        prefix: String,
        #[arg(long, default_value_t = 5000u32)]
        max: u32,
    },
    /// List kprobe symbols from kallsyms (Linux agent).
    Kprobe {
        #[arg(short, long, default_value = "")]
        prefix: String,
        #[arg(long, default_value_t = 5000u32)]
        max: u32,
    },
    /// List uprobe symbols from an ELF (path must exist on agent).
    Uprobe {
        #[arg(short, long)]
        binary: String,
        #[arg(short, long, default_value = "")]
        prefix: String,
        #[arg(long, default_value_t = 5000u32)]
        max: u32,
    },
    /// Send a local ELF to the agent and list section names.
    Inspect {
        #[arg(short, long)]
        file: String,
    },
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();
    let token = if args.token.is_empty() {
        None
    } else {
        Some(args.token.clone())
    };

    if let Some(cmd) = args.command {
        return run_discover(&args.agent, token, cmd).await;
    }

    run_repl(&args.agent, token, args.script_path.as_deref()).await
}

async fn run_discover(
    agent: &str,
    token: Option<String>,
    cmd: Commands,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut c = PhantomClient::connect(agent, token).await?;
    c.open_session("").await?;

    match cmd {
        Commands::Tracepoints { prefix, max } => {
            let names = c.list_tracepoints(&prefix, max).await?;
            for n in names {
                println!("{}", n);
            }
        }
        Commands::Kprobe { prefix, max } => {
            let syms = c.list_kprobe_symbols(&prefix, max).await?;
            for s in syms {
                println!("{}", s);
            }
        }
        Commands::Uprobe {
            binary,
            prefix,
            max,
        } => {
            let syms = c.list_uprobe_symbols(&binary, &prefix, max).await?;
            for s in syms {
                println!("{}", s);
            }
        }
        Commands::Inspect { file } => {
            let data = std::fs::read(&file)?;
            let secs = c.inspect_elf(&data).await?;
            for s in secs {
                println!("{}", s);
            }
        }
    }
    let _ = c.close_session().await;
    Ok(())
}

fn format_debug_event_line(ev: &DebugEvent) -> String {
    let ty = EventType::try_from(ev.event_type).unwrap_or(EventType::Unspecified);
    format!(
        "type={} timestamp_ns={} session_id={} pid={} tgid={} cpu={} probe_id={} payload_len={}",
        ty.as_str_name(),
        ev.timestamp_ns,
        ev.session_id,
        ev.pid,
        ev.tgid,
        ev.cpu,
        ev.probe_id,
        ev.payload.len(),
    )
}

async fn run_repl(
    agent: &str,
    token: Option<String>,
    script_path: Option<&str>,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut c = PhantomClient::connect(agent, token).await?;
    c.open_session("").await?;
    eprintln!("session {}", c.session_id());

    let mut stream_client = c.clone();
    tokio::spawn(async move {
        let Ok(mut stream) = stream_client.stream_events().await else {
            return;
        };
        loop {
            match stream.message().await {
                Ok(Some(ev)) => println!("{}", format_debug_event_line(&ev)),
                Ok(None) => break,
                Err(_) => break,
            }
        }
    });

    let stdin = io::stdin();
    let mut out = io::stdout();

    let interactive = script_path.is_none();
    let input: Box<dyn BufRead> = if let Some(path) = script_path {
        Box::new(io::BufReader::new(std::fs::File::open(path)?))
    } else {
        Box::new(stdin.lock())
    };

    let mut lines = input.lines();
    loop {
        if interactive {
            write!(out, "phantom> ")?;
            out.flush()?;
        }
        let line = match lines.next() {
            Some(Ok(l)) => l,
            Some(Err(e)) => return Err(e.into()),
            None => break,
        };
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let lower = line.to_lowercase();
        let first = lower.split_whitespace().next().unwrap_or("");
        if first == "quit" || first == "exit" || first == "q" {
            break;
        }
        match c.execute(line).await {
            Ok(resp) => match resp.into_result() {
                Ok(output) => {
                    if !output.is_empty() {
                        writeln!(out, "{}", output)?;
                    }
                }
                Err(msg) => {
                    writeln!(out, "{}", msg)?;
                    if !interactive {
                        let _ = c.close_session().await;
                        std::process::exit(1);
                    }
                }
            },
            Err(e) => {
                writeln!(out, "error: {}", e)?;
                if !interactive {
                    let _ = c.close_session().await;
                    std::process::exit(1);
                }
            }
        }
    }

    let _ = c.close_session().await;
    Ok(())
}
