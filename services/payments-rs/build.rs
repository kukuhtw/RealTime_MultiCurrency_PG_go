// services/payments-rs/build.rs
// services/payments-rs/build.rs
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let crate_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let repo_root = crate_dir.join("..").join("..");

    let include_dir    = repo_root.join("proto").join("gen");
    let payments_proto = include_dir.join("payments/v1/payments.proto");
    let db_proto       = include_dir.join("db/v1/db.proto");
    let common_proto   = include_dir.join("common/v1/common.proto");

    // Rebuild kalau ada perubahan di file ini
    println!("cargo:rerun-if-changed={}", payments_proto.display());
    println!("cargo:rerun-if-changed={}", db_proto.display());
    println!("cargo:rerun-if-changed={}", common_proto.display());
    println!("cargo:rerun-if-changed={}", include_dir.display());

    tonic_build::configure()
        .compile_protos(&[payments_proto, db_proto, common_proto], &[include_dir])?;

    Ok(())
}
