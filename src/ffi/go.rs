// use libc::c_char;
// use std::ffi::{CString, CStr};
//
// extern {
//     fn sendMessage(s: *const c_char);
//     fn receiveMessage() -> *mut c_char;
// }
//
// fn main(){
//     let rust_send_string = "Just a test";
//     let c_send_string = CString::new(rust_send_string)
//         .expect("CString::new failed");
//
//     unsafe{ sendMessage(c_send_string.as_ptr()); }
//
//     let c_recv_string: *mut c_char = unsafe {
//         receiveMessage()
//     };
//     let rust_recv_string = unsafe {
//         CStr::from_ptr(c_recv_string)
//             .to_string_lossy()
//             .into_owned()
//     };
//     println!("Received Message: {}", rust_recv_string);
// }

// extern "C" {
//     pub fn send_char(ptr: *const libc::c_char, len: libc::c_int);
// }
//
// #[no_mangle]
// pub extern "C" fn new_channel() -> *mut VecDeque<char>{
//     Box::into_raw(Box::new(VecDeque::new()))
// }
//
// // #[no_mangle]
// // pub extern "C" fn send_char(
// //     queue: *mut VecDeque<char>,
// //     ch: char
// // ){
// //     let queue = unsafe {
// //         &mut *queue
// //     };
// //     queue.push_back(ch);
// // }
//
// #[no_mangle]
// pub extern "C" fn receive_char(queue: *mut VecDeque<char>) -> char {
//     let queue = unsafe{ &mut *queue };
//     queue.pop_front().unwrap_or('\0')
// }
//
// #[no_mangle] extern "C" fn free_channel(queue: *mut VecDeque<char>) {
//     unsafe {
//         let _ = Box::from_raw(queue);
//     }
// }
//
// #[no_mangle] extern "C" fn send_utf8(
//     ptr: *const libc::c_char,
//     len: libc::c_int
// ){
//     let len = len as usize;
//     let slice = unsafe { std::slice::from_raw_parts(ptr as *const u8, len) };
//
//     // Received char from GO. Do with it as you will..
//     let received_char = std::str::from_utf8(slice)
//         .unwrap()
//         .chars()
//         .next()
//         .unwrap();
//
//     println!(" --> From Go: {}", received_char);
// }
