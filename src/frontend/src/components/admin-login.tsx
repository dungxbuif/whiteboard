import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import useAuthStore from "@/store/auth-store";

export function GoogleLogin() {
  const startOAuth = useAuthStore((state) => state.startOAuth);
  return (
    <main className="grid min-h-screen place-items-center bg-muted/30 p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Sign in to Whiteboard</CardTitle>
        </CardHeader>
        <CardContent>
          <Button className="w-full" onClick={startOAuth} type="button">
            Continue with Google
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
