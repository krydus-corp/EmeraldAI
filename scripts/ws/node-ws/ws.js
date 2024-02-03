/*
 * File: ws.js
 * Project: scripts
 * File Created: Tuesday, 9th August 2022 9:53:18 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */

import WebSocket from "ws";

const url = "wss://localhost/v1/uploads/62f28a2df68e582a7fb4a27b?websocket=true";
const token = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjYyZTc2NDBlY2IxYzAyN2NhY2FkNGQ2ZCIsInVpZCI6ImY0MDJlOTE4LTZhNDctNGViZi05NjVmLTNlYTRiMWJmZGRkYSIsImV4cCI6MTY2MDY2NjY3OX0.5_BzuOj_p-4Q6fXvDOyE5Vy4uiy_x2hGfHTO2UpJ4pg";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

const connection = new WebSocket(url, { headers: { "Content-Type": "application/json", Authorization: token } });

connection.on("message", function message(data) {
  console.log("received: %s", data);
});
