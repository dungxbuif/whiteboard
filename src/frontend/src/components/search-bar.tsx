import { Search } from "lucide-react";
import { useEffect, useRef } from "react";

import { Input } from "@/components/ui/input";
import useSearchStore, { SEARCH_DEBOUNCE_MS } from "@/store/search-store";

export function SearchBar({ workspaceId }: { workspaceId: string | null }) {
  const query = useSearchStore((state) => state.query);
  const view = useSearchStore((state) => state.view);
  const setQuery = useSearchStore((state) => state.setQuery);
  const search = useSearchStore((state) => state.search);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

  return (
    <label className="relative block w-80 max-w-full">
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} aria-hidden="true" />
      <Input
        aria-label="Search scenes and folders"
        className="pl-9"
        onChange={(event) => {
          const nextQuery = event.target.value;
          setQuery(nextQuery);
          if (timer.current) clearTimeout(timer.current);
          if (!workspaceId) return;
          timer.current = setTimeout(() => {
            void search(workspaceId, nextQuery, view).catch(() => undefined);
          }, SEARCH_DEBOUNCE_MS);
        }}
        placeholder="Search scenes and folders"
        value={query}
      />
    </label>
  );
}
