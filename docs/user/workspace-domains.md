# Verify a workspace domain

Workspace owners can prove control of a domain and explicitly capture it for future workspace sign-in settings.

1. Open **Workspace settings → Identity & domains**.
2. Enter the exact domain, such as `example.com`, and choose **Add domain**.
3. Copy the displayed TXT name and value into your public DNS provider. The challenge expires after 24 hours and is shown only once.
4. Wait for DNS publication, then choose **Verify DNS**.
5. Review the result and choose **Accept and capture**.

The application checks only `_whiteboard-verification.<your-domain>`. A parent domain does not verify a subdomain, and a subdomain does not verify its parent. Internationalized names are shown and stored in normalized ASCII form.

Domain proof does not link people by email, merge accounts, change memberships, or move scenes and files. A current workspace owner must separately accept the capture while authenticated. Use **Revoke** to disable a captured domain.

If verification fails, confirm that the complete TXT value is on the exact generated name. DNS caches can delay visibility. Create a new challenge when the current one has expired.
