import { handleRequest, type ProxyEnv } from "./registry";

export default {
  fetch(request: Request, env: ProxyEnv): Promise<Response> {
    return handleRequest(request, env);
  },
};
