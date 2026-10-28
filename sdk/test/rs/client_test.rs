use arbeiter_auth_sdk::AuthClient;
use std::io::{Read, Write};
use std::net::TcpListener;
use std::thread;

fn mock_server() -> (String, thread::JoinHandle<Vec<String>>) {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let address = format!("http://{}", listener.local_addr().unwrap());
    let handle = thread::spawn(move || {
        let mut requests = Vec::new();
        for index in 0..2 {
            let (mut stream, _) = listener.accept().unwrap();
            let mut data = Vec::new();
            let mut buffer = [0u8; 2048];
            loop {
                let n = stream.read(&mut buffer).unwrap();
                if n == 0 { break; }
                data.extend_from_slice(&buffer[..n]);
                if data.windows(4).any(|window| window == b"\r\n\r\n") { break; }
            }
            requests.push(String::from_utf8_lossy(&data).to_string());
            let payload = if index == 0 { r#"{"id":"u1"}"# } else { r#"{"email":"user@example.test"}"# };
            let cookie = if index == 0 { "Set-Cookie: session=opaque; Path=/; HttpOnly\r\n" } else { "" };
            write!(stream,"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n{cookie}Content-Length: {}\r\nConnection: close\r\n\r\n{payload}",payload.len()).unwrap();
        }
        requests
    });
    (address, handle)
}

#[test]
fn retains_session_cookie_and_decodes_user() {
    let (base, server) = mock_server();
    let client = AuthClient::new(base).unwrap();
    client.sign_in("user@example.test","password").unwrap();
    let user = client.me().unwrap().unwrap();
    assert_eq!(user["email"],"user@example.test");
    let requests=server.join().unwrap();
    assert!(requests[1].to_ascii_lowercase().contains("cookie: session=opaque"));
}
