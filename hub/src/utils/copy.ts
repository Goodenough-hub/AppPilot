export async function copyText(s: string): Promise<void> {
  await navigator.clipboard.writeText(s)
}