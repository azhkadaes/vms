import { createServer } from "node:http";
import { previewApi } from "../../tooling/preview.ts";

export async function startPreview({ time = "2099-01-01T08:00:00+08:00", session } = {}) {
  let current = Date.parse(time);
  let handler;
  let prefix;
  previewApi({ now: () => current, session }).configureServer({
    middlewares: {
      use(path, middleware) {
        prefix = path;
        handler = middleware;
      },
    },
  });
  const server = createServer((req, res) => {
    if (!req.url.startsWith(prefix + "/")) {
      res.writeHead(404).end();
      return;
    }
    req.url = req.url.slice(prefix.length);
    void handler(req, res, () => res.writeHead(404).end());
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  return {
    base: `http://127.0.0.1:${server.address().port}${prefix}`,
    setNow(time) {
      current = typeof time === "number" ? time : Date.parse(time);
    },
    async close() {
      server.closeAllConnections();
      await new Promise((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    },
  };
}
