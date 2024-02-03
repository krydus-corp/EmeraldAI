"""
File: ws.py
Project: scripts
File Created: Thursday, 16th June 2022 4:35:23 pm
Author: Anonymous (anonymous@gmail.com)
-----
Last Modified: Friday, 2nd February 2024 2:30:15 pm
Modified By: Anonymous (anonymous@gmail.com>)

Run via |
python3 scripts/ws.py --address 'wss://localhost/v1/uploads/{id}?websocket=true&Authorization={token}'
python3 scripts/ws.py --address 'wss://localhost/v1/exports/{id}?websocket=true&Authorization={token}'
python3 scripts/ws.py --address 'wss://localhost/v1/models/{id}?websocket=true&Authorization={token}'
"""

import argparse
import asyncio
import ssl
import time

import websockets

ssl_context = ssl.SSLContext()
ssl_context.check_hostname = False
ssl_context.verify_mode = ssl.CERT_NONE


async def run(uri: str):
    async with websockets.connect(
        uri, ssl=ssl_context, ping_interval=None
    ) as websocket:
        while True:
            try:
                greeting = await websocket.recv()
                print("< {}".format(greeting))
            except Exception as e:
                print(e)
                break

            time.sleep(1)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--address", type=str, required=True, help="address")

    args = parser.parse_args()

    asyncio.run(run(args.address))
