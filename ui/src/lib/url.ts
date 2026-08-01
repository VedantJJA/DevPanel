/**
 * Shared utility for building absolute project URLs across path and subdomain routing modes.
 */
export function buildProjectUrl(opts: {
  routingMode: "path" | "subdomain";
  rootDomain: string;
  projectName: string;
  path?: string; // optional sub-path inside the project, e.g. "/foo/bar"
}): string {
  const safeName = encodeURIComponent(opts.projectName);
  const path = opts.path ?? "";
  const cleanPath = path ? (path.startsWith("/") ? path : `/${path}`) : "";
  if (opts.routingMode === "subdomain") {
    return `https://${safeName}.${opts.rootDomain}${cleanPath}`;
  }
  return `https://${opts.rootDomain}/app/${safeName}${cleanPath}`;
}
