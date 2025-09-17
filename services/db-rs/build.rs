// services/db-rs/build.rs
// 

fn main() {
    tonic_build::configure()
        .compile(
            &["../../proto/db/v1/db.proto"],   // sesuaikan path proto kamu
            &["../../proto"],                  // root proto dir
        )
        .expect("proto compile failed");
}

