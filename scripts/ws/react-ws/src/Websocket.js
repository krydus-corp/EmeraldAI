/*
 * File: Websocket.js
 * Project: ws
 * File Created: Monday, 5th September 2022 8:28:58 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
import React from "react";

const Websocket = () => {
  function initiateWebsocket() {
    const client = new WebSocket("wss://localhost/v1/uploads/aa468146-17ff-11ed-ae6c-8bb804211226?client_id=TRUE&websocket=TRUE&Authorization=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjYzMTAzNDI4NTMyNTE4MWU0MDU4MjE0MCIsInVpZCI6ImU5ZjY3YzM5LWU0ZDYtNGU2Yi04ZDFlLTU3MGM0ODI4OWM3MSIsImV4cCI6MTY2MjQwMTMzNH0.WIgGcta80-c11YXnjz7oYamvYK6--BsRsLgxkmFkNcw");

    client.onopen = () => {
      console.log("WebSocket Client Connected");
    };
    client.onmessage = (message) => {
      console.log(message);
    };
  }

  return <button onClick={initiateWebsocket}>Initiate Websocket!</button>;
};

export default Websocket;
