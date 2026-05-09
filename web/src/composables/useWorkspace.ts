export async function fetchWorkspaceFiles(): Promise<string[]> {
  const r = await fetch("/api/workspace/files");
  if (!r.ok) throw new Error(`列出文件失败: ${r.status}`);
  const j = (await r.json()) as { files?: string[] };
  return j.files ?? [];
}

export async function fetchWorkspaceFile(path: string): Promise<string> {
  const r = await fetch("/api/workspace/file?" + new URLSearchParams({ path }));
  if (!r.ok) throw new Error(`读取失败: ${r.status}`);
  const j = (await r.json()) as { content?: string };
  return j.content ?? "";
}

export async function putWorkspaceFile(path: string, content: string): Promise<void> {
  const r = await fetch("/api/workspace/file", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, content }),
  });
  if (!r.ok) throw new Error(`保存失败: ${r.status}`);
}
