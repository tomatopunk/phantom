fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_dir = std::path::Path::new("../proto");
    tonic_build::configure()
        .build_server(false)
        .compile_protos(&[proto_dir.join("debugger.proto")], &[proto_dir])?;
    Ok(())
}
