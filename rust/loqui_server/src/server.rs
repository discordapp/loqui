use crate::connection_handler::ConnectionHandler;
use crate::{Config, RequestHandler};
use failure::Error;
use loqui_connection::{Connection, Factory};
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::net::{TcpListener, TcpStream};
use tokio::prelude::*;

pub struct Server<R: RequestHandler<F>, F: Factory> {
    config: Arc<Config<R, F>>,
}

impl<R: RequestHandler<F>, F: Factory> Server<R, F> {
    pub fn new(config: Config<R, F>) -> Self {
        Self {
            config: Arc::new(config),
        }
    }

    fn handle_connection(&self, tcp_stream: TcpStream) {
        let connection_handler = ConnectionHandler::new(self.config.clone());
        let _connection = Connection::spawn(tcp_stream, connection_handler, None);
    }

    pub async fn listen_and_serve(&self, address: String) -> Result<(), Error> {
        let addr: SocketAddr = address.parse()?;
        let listener = TcpListener::bind(&addr)?;
        info!("Starting {:?} ...", address);
        let mut incoming = listener.incoming();
        loop {
            match await!(incoming.next()) {
                Some(Ok(tcp_stream)) => {
                    self.handle_connection(tcp_stream);
                }
                other => {
                    println!("incoming.next() failed. {:?}", other);
                }
            }
        }
    }
}
