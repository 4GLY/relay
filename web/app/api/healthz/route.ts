// Static, dependency-free health endpoint for Kubernetes liveness/readiness
// probes. Keeps the probe decoupled from any future SSR fetch the homepage
// might add.
export const dynamic = "force-static";

export function GET() {
  return new Response("ok", {
    status: 200,
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}
