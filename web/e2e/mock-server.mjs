import http from "node:http"

http
  .createServer((request, response) => {
    response.setHeader("Content-Type", "application/json")
    response.end(JSON.stringify({ code: 0, errcode: 0, path: request.url }))
  })
  .listen(19090, "127.0.0.1")
