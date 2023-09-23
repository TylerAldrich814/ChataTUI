
use anyhow::{Result, anyhow};
use serde::Deserialize;
use reqwest;
use chatatui_lib::run_application;

#[derive(Deserialize)]
struct HomeMessage {
  #[serde(rename="authed")]
  authed: bool,
  #[serde(rename="token")]
  token:  String,
}

#[tokio::main(flavor="multi_thread")]
async fn main() -> Result<()> {
  run_application().await?;
  
  println!("HERE");
  Ok(())
}