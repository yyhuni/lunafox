# 独立部署 LunaFox AI Proxy 指南

建议创建一个**独立的 GitHub 仓库**来部署这个代理服务，这样可以避免路径配置错误，且更易于维护。

## 1. 准备代码

在你的电脑上找一个空文件夹（不要在 lunafox 项目里），新建一个 `main.ts` 文件，内容如下：

```typescript
const OPENAI_API_KEY = Deno.env.get("AI_API_KEY");
const OPENAI_BASE_URL = Deno.env.get("AI_BASE_URL") || "https://api.deepseek.com/v1";

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type",
};

Deno.serve(async (req) => {
  if (req.method === "OPTIONS") {
    return new Response(null, { headers: CORS_HEADERS });
  }

  if (req.method !== "POST") {
    return new Response("Only POST is allowed", { status: 405, headers: CORS_HEADERS });
  }

  try {
    if (!OPENAI_API_KEY) {
      return new Response(JSON.stringify({ error: "Server AI_API_KEY not configured" }), {
        status: 500,
        headers: { "Content-Type": "application/json", ...CORS_HEADERS },
      });
    }

    const { context } = await req.json();

    const prompt = `
You are LunaFox (月狐), a witty, humorous, hacker-culture-savvy virtual assistant for cybersecurity professionals.
Current context:
- Hour: ${context?.hour}
- Day: ${context?.day} (0=Sun, 5=Fri)
- Event: ${context?.event || "idle"}

Task: Generate a single JSON object for a toast notification.
Format:
{
  "title": "Short title (<10 chars)",
  "description": "One sentence message (<30 chars), hacker style, maybe funny or warm.",
  "icon": "A single emoji representing the mood",
  "primaryAction": { "label": "Button Text" },
  "secondaryAction": { "label": "Cancel Text" }
}

Do NOT output markdown. Output ONLY the JSON string.
`;

    const aiResponse = await fetch(`${OPENAI_BASE_URL}/chat/completions`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${OPENAI_API_KEY}`,
      },
      body: JSON.stringify({
        model: "deepseek-chat",
        messages: [{ role: "system", content: prompt }],
        response_format: { type: "json_object" },
      }),
    });

    if (!aiResponse.ok) {
      const err = await aiResponse.text();
      throw new Error(`AI API Error: ${err}`);
    }

    const aiData = await aiResponse.json();
    const content = aiData.choices[0].message.content;
    const jsonContent = JSON.parse(content);

    return new Response(JSON.stringify(jsonContent), {
      headers: { "Content-Type": "application/json", ...CORS_HEADERS },
    });
  } catch (err: any) {
    console.error(err);
    return new Response(JSON.stringify({ error: "Failed to generate nudge", details: err.message }), {
      status: 500,
      headers: { "Content-Type": "application/json", ...CORS_HEADERS },
    });
  }
});
```

## 2. 提交到新仓库

```bash
# 在该文件夹下
git init
git add main.ts
git commit -m "init proxy"
# 去 GitHub 创建一个新仓库 (例如 lunafox-proxy)，然后关联并推送
git remote add origin https://github.com/YOUR_USER/lunafox-proxy.git
git push -u origin main
```

## 3. 在 Deno Deploy 部署

1.  访问 [Deno Dash](https://dash.deno.com)。
2.  点击 **New Project**。
3.  选择刚才创建的 `lunafox-proxy` 仓库。
4.  它应该会自动识别 `main.ts`。点击 **Link**。
5.  部署成功后，去 **Settings -> Environment Variables** 添加：
    *   `AI_API_KEY`: 你的 DeepSeek Key (`sk-xxxx`)。
    *   `AI_BASE_URL`: `https://api.deepseek.com/v1`。

## 4. 回到 LunaFox 主项目

在 `frontend/.env` 中填入你获得的 Deno 域名：

```env
NEXT_PUBLIC_NUDGE_API_URL=https://your-project.deno.dev
```
