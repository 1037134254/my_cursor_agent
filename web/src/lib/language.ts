export function pathToMonacoLanguage(path: string): string {
  const i = path.lastIndexOf(".");
  const ext = i >= 0 ? path.slice(i + 1).toLowerCase() : "";
  const map: Record<string, string> = {
    ts: "typescript",
    tsx: "typescript",
    js: "javascript",
    jsx: "javascript",
    mjs: "javascript",
    cjs: "javascript",
    go: "go",
    json: "json",
    md: "markdown",
    html: "html",
    htm: "html",
    vue: "html",
    css: "css",
    scss: "scss",
    less: "less",
    yaml: "yaml",
    yml: "yaml",
    xml: "xml",
    sh: "shell",
    bash: "shell",
    py: "python",
    rs: "rust",
    toml: "ini",
  };
  return map[ext] ?? "plaintext";
}
