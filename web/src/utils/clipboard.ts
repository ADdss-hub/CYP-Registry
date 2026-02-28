export async function copyToClipboard(text: string): Promise<void> {
  if (!text && text !== "") {
    // 空值直接返回，不做复制
    return;
  }

  // 优先使用现代 Clipboard API，并增加文档焦点判断，避免 "Document is not focused" 报错
  try {
    if (
      typeof navigator !== "undefined" &&
      typeof window !== "undefined" &&
      navigator.clipboard &&
      (window as any).isSecureContext !== false
    ) {
      // 一些浏览器在 document 失焦时会拒绝写入剪贴板
      if (
        typeof document !== "undefined" &&
        typeof document.hasFocus === "function"
      ) {
        if (!document.hasFocus()) {
          throw new Error("document-not-focused");
        }
      }

      await navigator.clipboard.writeText(text);
      return;
    }
  } catch {
    // 继续走降级方案
  }

  // 兼容性降级方案：使用隐藏 textarea + execCommand('copy')
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.left = "-9999px";
  textarea.style.top = "0";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const ok = document.execCommand("copy");
  document.body.removeChild(textarea);

  if (!ok) {
    throw new Error("fallback-copy-failed");
  }
}
