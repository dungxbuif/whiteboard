import { useEffect, useState } from "react";

export function navigate(path: string) {
  if (`${window.location.pathname}${window.location.search}` === path) return;
  window.history.pushState({}, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

export function useLocation() {
  const [location, setLocation] = useState(() => ({ pathname: window.location.pathname, search: window.location.search }));
  useEffect(() => {
    const handlePopState = () => setLocation({ pathname: window.location.pathname, search: window.location.search });
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);
  return location;
}

export function usePathname() {
  const [pathname, setPathname] = useState(() => window.location.pathname);

  useEffect(() => {
    const handlePopState = () => setPathname(window.location.pathname);
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  return pathname;
}
