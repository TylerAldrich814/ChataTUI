
const BASE_URL: &str = "http://127.0.0.1:8080/";
const WS_URL: &str = "ws://127.0.0.1:8080/";

const SIGN_IN: &str = "/User/Signin";
const SIGN_UP: &str = "/User/Signup";

const CHATROOMS: &str = "/chatrooms";
const CHATROOM_BY_ID: &str = "/chatrooms/{}";

const CHATROOM_WS: &str = "/chatrooms/{}/ws";


pub fn post_sign_in_url() -> String {
    format!("{}{}", BASE_URL, SIGN_IN)
}

pub fn post_sign_up_url() -> String {
    format!("{}{}", BASE_URL, SIGN_UP)
}

pub fn get_chatrooms_url() -> String {
    format!("{}{}", BASE_URL, CHATROOMS)
}

pub fn get_chatroom_by_id_url(room_id: &str) -> String {
    format!("{}{}", BASE_URL, CHATROOM_BY_ID.replace("{}", room_id))
}

pub fn get_chatroom_join_url(room_id: &str) -> String {
    format!("{}{}/join", BASE_URL, CHATROOM_BY_ID.replace("{}", room_id))
}
pub fn get_chatroom_get_messages(room_id: &str) -> String {
    format!("{}{}/messages", BASE_URL, CHATROOM_BY_ID.replace("{}", room_id))
}

pub fn chatroom_on_load(room_id: &str) -> String {
    format!("{}{}/load", BASE_URL, CHATROOM_BY_ID.replace("{}", room_id))
}
pub fn chatroom_ws_url(room_id: &str) -> String {
    format!("{}{}", WS_URL, CHATROOM_WS.replace("{}", room_id))
}