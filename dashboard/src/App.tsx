import { Folder, LayoutDashboard, Plus, Search, Star } from "lucide-react";

import { useDashboardStore } from "./store/dashboard-store";

const workspaces = ["Design Ops", "Product Boards", "Engineering"];
const recentScenes = [
  { title: "Launch flow", folder: "Product Boards", updated: "Today" },
  { title: "Auth rewrite", folder: "Engineering", updated: "Yesterday" },
  { title: "Dashboard IA", folder: "Design Ops", updated: "Jun 3" },
];

function App() {
  const activeWorkspace = useDashboardStore((state) => state.activeWorkspace);
  const setActiveWorkspace = useDashboardStore((state) => state.setActiveWorkspace);

  return (
    <main className="dashboard-shell">
      <aside className="sidebar" aria-label="Workspace navigation">
        <div className="brand">
          <LayoutDashboard size={22} aria-hidden="true" />
          <span>Devtool</span>
        </div>

        <nav className="workspace-list" aria-label="Workspaces">
          {workspaces.map((workspace) => (
            <button
              className={workspace === activeWorkspace ? "workspace active" : "workspace"}
              key={workspace}
              onClick={() => setActiveWorkspace(workspace)}
              type="button"
            >
              <Folder size={16} aria-hidden="true" />
              <span>{workspace}</span>
            </button>
          ))}
        </nav>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <p className="eyebrow">Workspace</p>
            <h1>{activeWorkspace}</h1>
          </div>

          <div className="actions">
            <label className="search">
              <Search size={16} aria-hidden="true" />
              <input aria-label="Search scenes" placeholder="Search scenes" />
            </label>
            <button className="primary-action" type="button">
              <Plus size={16} aria-hidden="true" />
              <span>New scene</span>
            </button>
          </div>
        </header>

        <div className="scene-grid">
          {recentScenes.map((scene) => (
            <article className="scene-card" key={scene.title}>
              <div className="scene-card-header">
                <Folder size={18} aria-hidden="true" />
                <Star size={16} aria-label="Not starred" />
              </div>
              <h2>{scene.title}</h2>
              <p>{scene.folder}</p>
              <span>{scene.updated}</span>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}

export default App;
