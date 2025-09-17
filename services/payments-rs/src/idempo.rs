// services/payments-rs/src/idempo.rs
use moka::future::Cache;
use std::time::Duration;

#[derive(Clone)]
pub struct IdempoCache {
    inner: Cache<String, String>, // key -> compact response (json/status)
}

impl IdempoCache {
    pub fn new() -> Self {
        let inner = Cache::builder()
            .time_to_live(Duration::from_secs(10 * 60))
            .max_capacity(100_000)
            .build();
        Self { inner }
    }

    pub async fn get(&self, key: &str) -> Option<String> {
        self.inner.get(key).await
    }
    pub async fn put(&self, key: String, val: String) {
        self.inner.insert(key, val).await;
    }
}
