# Verify a workspace domain

Workspace owners can prove control of a domain and explicitly capture it for future workspace sign-in settings.

1. Open **Workspace settings → Identity & domains**.
2. Enter the exact domain, such as `example.com`, and choose **Add domain**.
3. Copy the displayed TXT name and value into your public DNS provider. The challenge expires after 24 hours and is shown only once.
4. Wait for DNS publication, then choose **Verify DNS**.
5. Keep the exact TXT record published, review the result, and choose **Accept and capture** before the 24-hour challenge expires. The application checks DNS again during capture.

The application checks only `_whiteboard-verification.<your-domain>`. A parent domain does not verify a subdomain, and a subdomain does not verify its parent. Internationalized names are shown and stored in normalized ASCII form.

Domain proof does not link people by email, merge accounts, change memberships, or move scenes and files. A current workspace owner must separately accept the capture while authenticated. Use **Revoke** to disable a captured domain.

If verification or capture fails, confirm that the complete TXT value remains on the exact generated name. A failed capture leaves the domain verified so you can restore DNS and retry before expiry. DNS caches can delay visibility. Create a new challenge when the current one has expired. An expired pending claim can be claimed by another workspace; verified and captured claims remain protected.

The settings screen safely retries a request that timed out without a response. It reuses the same request ID, so a challenge already committed by the server is returned again with the same TXT value. A definite server response or changing the requested domain ends that retry attempt and the next action uses a new request ID.
