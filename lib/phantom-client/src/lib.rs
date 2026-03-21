//! Shared [tonic] gRPC client for the Phantom debugger agent.

pub mod phantom {
    pub mod api {
        tonic::include_proto!("phantom.api");
    }
}

pub use phantom::api::{
    debugger_service_client::DebuggerServiceClient, CloseSessionRequest, CompileAndAttachRequest,
    CompileAndAttachResponse, DebugEvent, ExecuteRequest, ExecuteResponse, GetHostMetricsRequest,
    GetHostMetricsResponse, GetTaskTreeRequest, GetTaskTreeResponse, InspectElfRequest,
    ListKprobeSymbolsRequest, ListSessionsRequest, ListTracepointsRequest,
    ListUprobeSymbolsRequest, OpenSessionRequest, OpenSessionResponse, StreamEventsRequest,
};

use tonic::metadata::AsciiMetadataValue;
use tonic::Request;

/// High-level client: session, optional Bearer token, and helpers for common RPCs.
pub struct PhantomClient {
    inner: DebuggerServiceClient<tonic::transport::Channel>,
    token: Option<String>,
    session_id: String,
}

impl PhantomClient {
    /// Connects to the agent. `agent` may be `host:port` or a full `http://` URL.
    pub async fn connect(
        agent: &str,
        token: Option<String>,
    ) -> Result<Self, tonic::transport::Error> {
        let url = if agent.starts_with("http://") || agent.starts_with("https://") {
            agent.to_string()
        } else {
            format!("http://{}", agent)
        };
        let inner = DebuggerServiceClient::connect(url).await?;
        Ok(Self {
            inner,
            token,
            session_id: String::new(),
        })
    }

    fn with_auth<T>(&self, mut req: Request<T>) -> Request<T> {
        if let Some(ref t) = self.token {
            let v = format!("Bearer {}", t.trim());
            if let Ok(val) = AsciiMetadataValue::try_from(v.as_str()) {
                req.metadata_mut().insert("authorization", val);
            }
        }
        req
    }

    /// Opens or resumes a session (empty `session_id` lets the server assign one).
    pub async fn open_session(&mut self, session_id: &str) -> Result<String, tonic::Status> {
        let resp = self
            .inner
            .open_session(self.with_auth(Request::new(OpenSessionRequest {
                session_id: session_id.to_string(),
            })))
            .await?
            .into_inner();
        self.session_id = resp.session_id.clone();
        Ok(resp.session_id)
    }

    pub fn session_id(&self) -> &str {
        &self.session_id
    }

    pub async fn execute(&mut self, command_line: &str) -> Result<ExecuteResponse, tonic::Status> {
        if self.session_id.is_empty() {
            return Err(tonic::Status::failed_precondition(
                "not connected: call open_session first",
            ));
        }
        self.inner
            .execute(self.with_auth(Request::new(ExecuteRequest {
                session_id: self.session_id.clone(),
                command_line: command_line.to_string(),
            })))
            .await
            .map(|r| r.into_inner())
    }

    pub async fn stream_events(&mut self) -> Result<tonic::Streaming<DebugEvent>, tonic::Status> {
        if self.session_id.is_empty() {
            return Err(tonic::Status::failed_precondition("not connected"));
        }
        self.inner
            .stream_events(self.with_auth(Request::new(StreamEventsRequest {
                session_id: self.session_id.clone(),
            })))
            .await
            .map(|r| r.into_inner())
    }

    /// Host-wide /proc metrics; does not require an open session.
    pub async fn get_host_metrics(&mut self) -> Result<GetHostMetricsResponse, tonic::Status> {
        self.inner
            .get_host_metrics(self.with_auth(Request::new(GetHostMetricsRequest {})))
            .await
            .map(|r| r.into_inner())
    }

    /// Lists tasks under `/proc/<tgid>/task`; does not require an open session.
    pub async fn get_task_tree(&mut self, tgid: u32) -> Result<GetTaskTreeResponse, tonic::Status> {
        self.inner
            .get_task_tree(self.with_auth(Request::new(GetTaskTreeRequest { tgid })))
            .await
            .map(|r| r.into_inner())
    }

    pub async fn list_tracepoints(
        &mut self,
        prefix: &str,
        max_entries: u32,
    ) -> Result<Vec<String>, tonic::Status> {
        let r = self
            .inner
            .list_tracepoints(self.with_auth(Request::new(ListTracepointsRequest {
                prefix: prefix.to_string(),
                max_entries,
            })))
            .await?
            .into_inner();
        Ok(r.names)
    }

    pub async fn list_kprobe_symbols(
        &mut self,
        prefix: &str,
        max_entries: u32,
    ) -> Result<Vec<String>, tonic::Status> {
        let r = self
            .inner
            .list_kprobe_symbols(self.with_auth(Request::new(ListKprobeSymbolsRequest {
                prefix: prefix.to_string(),
                max_entries,
            })))
            .await?
            .into_inner();
        Ok(r.symbols)
    }

    pub async fn list_uprobe_symbols(
        &mut self,
        binary_path: &str,
        prefix: &str,
        max_entries: u32,
    ) -> Result<Vec<String>, tonic::Status> {
        let r = self
            .inner
            .list_uprobe_symbols(self.with_auth(Request::new(ListUprobeSymbolsRequest {
                binary_path: binary_path.to_string(),
                prefix: prefix.to_string(),
                max_entries,
            })))
            .await?
            .into_inner();
        Ok(r.symbols)
    }

    pub async fn inspect_elf(&mut self, elf_data: &[u8]) -> Result<Vec<String>, tonic::Status> {
        let r = self
            .inner
            .inspect_elf(self.with_auth(Request::new(InspectElfRequest {
                elf_data: elf_data.to_vec(),
            })))
            .await?
            .into_inner();
        Ok(r.section_names)
    }

    pub async fn compile_and_attach(
        &mut self,
        source: &str,
        attach: &str,
        program_name: &str,
    ) -> Result<CompileAndAttachResponse, tonic::Status> {
        if self.session_id.is_empty() {
            return Err(tonic::Status::failed_precondition("not connected"));
        }
        self.inner
            .compile_and_attach(self.with_auth(Request::new(CompileAndAttachRequest {
                session_id: self.session_id.clone(),
                source: source.to_string(),
                attach: attach.to_string(),
                program_name: program_name.to_string(),
            })))
            .await
            .map(|r| r.into_inner())
    }

    pub async fn close_session(&mut self) -> Result<(), tonic::Status> {
        if self.session_id.is_empty() {
            return Ok(());
        }
        let sid = self.session_id.clone();
        self.inner
            .close_session(self.with_auth(Request::new(CloseSessionRequest { session_id: sid })))
            .await?;
        self.session_id.clear();
        Ok(())
    }
}
