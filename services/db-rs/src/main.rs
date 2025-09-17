// services/db-rs/src/main.rs
use std::net::SocketAddr;

use tonic::transport::Server;
use tracing::info;

mod handlers;
mod store; // db logic

pub mod dbv1 {
    tonic::include_proto!("db.v1");
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt().with_env_filter("info").init();

    // ENV
    let db_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://postgres:secret@localhost:5432/poc".into());
    let grpc_addr: SocketAddr = "0.0.0.0:9095".parse().unwrap();
    let metrics_addr: SocketAddr = "0.0.0.0:9105".parse().unwrap();

    // PgPool
    let pool = sqlx::postgres::PgPoolOptions::new()
        .max_connections(16)
        .connect(&db_url)
        .await?;

    // /metrics sederhana
    tokio::spawn(async move {
        use hyper::{service::service_fn, Body, Request, Response, Server as HServer};
        use prometheus::{Encoder, TextEncoder};
        let registry = prometheus::Registry::new();
        let make = hyper::service::make_service_fn(move |_| {
            let registry = registry.clone();
            async move {
                Ok::<_, hyper::Error>(service_fn(move |_req: Request<Body>| {
                    let registry = registry.clone();
                    async move {
                        let mut buf = Vec::new();
                        let enc = TextEncoder::new();
                        enc.encode(&registry.gather(), &mut buf).unwrap();
                        Ok::<_, hyper::Error>(Response::new(Body::from(buf)))
                    }
                }))
            }
        });
        let _ = HServer::bind(&metrics_addr).serve(make).await;
    });

    let svc = handlers::DbService::new(pool);

    info!("db-rs gRPC on {}", grpc_addr);
    Server::builder()
        .add_service(dbv1::db_server::DbServer::new(svc))
        .serve(grpc_addr)
        .await?;

    Ok(())
}
