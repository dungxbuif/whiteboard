import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import useAuthStore from "@/store/auth-store";

export function PasswordLogin() {
  const login = useAuthStore((state) => state.login);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await login(email, password);
    } catch {
      setError("Invalid email or password");
    }
  }

  return (
    <main className="grid min-h-screen place-items-center bg-muted/30 p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Sign in to Whiteboard</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="grid gap-4" onSubmit={handleSubmit}>
            <label className="grid gap-2 text-sm font-medium" htmlFor="email">
              Email
              <Input autoComplete="email" id="email" onChange={(event) => setEmail(event.target.value)} type="email" value={email} />
            </label>
            <label className="grid gap-2 text-sm font-medium" htmlFor="password">
              Password
              <Input autoComplete="current-password" id="password" onChange={(event) => setPassword(event.target.value)} type="password" value={password} />
            </label>
            {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
            <Button className="w-full" type="submit">Sign in</Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
