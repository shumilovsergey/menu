# auth-center — fix: delegate exchange drops the user's name & provider

> Spec for a change in **auth-center** (`auth-center.sh-development.ru`). Nothing in
> `menu` or `wgetbash` needs to change if this is done as described — that's the point.

## Symptom

- Log into **menu** directly with Google → profile shows `GOOGLE` / `Сергей Шумилов` / `105993223363753461270`. ✅
- From menu, open **wgetbash** via the delegate redirect → auto-login works, `auth_id` is
  correct (`105993223363753461270`), but the profile shows `DELEGATE` / **`no name`**. ❌

The identity is right; the **display name and the real provider are lost** on the delegated
session only.

## Root cause

Apps don't invent the name — they render exactly what auth-center returns from
`POST /exchange`. menu's parser (`build/auth-human.go`) expects this shape:

```jsonc
{
  "ok": true,
  "method": "google",              // the identity provider
  "user": { "id": "1059...", "name": "Сергей Шумилов", "first_name": "...", "last_name": "..." },
  "error": ""
}
```

and derives the display name like this (`extractName`, auth-human.go:156):

```go
if method == "google" {
    return user["name"]            // Google → user.name
}
// otherwise join user.first_name + user.last_name   (Telegram / Solana style)
```

On a **direct** Google login auth-center returns `method:"google"` + `user.name`, so it works.

On a **delegate** exchange auth-center instead returns `method:"delegate"` with a `user`
object that has **no name fields**. Two things then go wrong on the client, and *both* stem
from auth-center:

1. `method` is `"delegate"`, so even if `user.name` were present the client wouldn't read it
   (the `name` path is gated on `method == "google"`).
2. The `user` payload carries no `name` / `first_name` / `last_name` at all → empty → `no name`.

So this is **not** a menu or wgetbash bug. They faithfully display the exchange payload;
menu only looks correct because it never went through delegation.

## The fix (in auth-center)

**Make a delegate exchange return the user's real identity, identical in shape to a normal
first-party login for that user.** auth-center is the identity source — it already has the
delegating user's stored profile (provider + name) keyed by `auth_id` — it just needs to echo
it back instead of a synthetic "delegate" identity.

### 1. At `/delegate` (code creation)

menu calls it with only `{ "user_id": "<auth_id>", "app_token": "..." }` (see
`menu/build/auth-server.go:delegateCode`). auth-center already knows `user_id`, so when it
mints the one-time code it should associate that code with the user's stored
**original provider** and **profile fields** (or just store `user_id` and re-resolve them at
exchange time — either works).

### 2. At `/exchange` (code redemption) — the actual change

When the submitted `code` is a delegate code, respond with the **original** provider and
profile, not `method:"delegate"` + empty user:

```jsonc
{
  "ok": true,
  "method": "google",                         // ← original provider, NOT "delegate"
  "user": {
    "id": "105993223363753461270",
    "name": "Сергей Шумилов",                 // ← carry the stored name
    "first_name": "Сергей",                    //   (include first/last too, for non-Google providers)
    "last_name": "Шумилов"
  },
  "error": ""
}
```

With this, every downstream app (menu, wgetbash, future apps) shows the correct name and
provider through the existing template code — **zero client changes required**.

### Why `method` must be the real provider

`method` is consumed as the identity provider (it gates the name logic and is shown as the
`GOOGLE` label). `"delegate"` is a *transport* detail, not an identity — overloading `method`
with it is what breaks the name path and mislabels the provider. Return the true provider.

## Optional: keep an audit trail of delegation

If you want to record that a session arrived via delegation (useful for logs/security), don't
reuse `method` for it — add a separate, non-breaking field, e.g.:

```jsonc
{ "ok": true, "method": "google", "via": "delegate", "user": { ... } }
```

`via` is ignored by current apps (they don't read it), so it stays backward compatible. Apps
can later surface a small "opened via menu" hint if desired.

## Acceptance check

1. Log into menu with Google.
2. Open wgetbash from menu's grid (delegate redirect).
3. wgetbash profile popover shows `GOOGLE` (or original provider) + `Сергей Шумилов` +
   `105993223363753461270` — matching what menu shows.
4. Confirm no code changes were needed in menu or wgetbash.

## Notes / edge cases

- **Non-Google providers:** include `first_name` / `last_name` in the delegate `user` payload
  so the join path works (Telegram/Solana don't set `user.name`).
- **auth_id stays canonical:** apps key their `users` table on `auth_id` (`menu/build/db.go`),
  so fixing only the name/provider won't disturb existing rows — `upsertUser` updates the
  name on next login when a non-empty name finally arrives (`db.go:upsertUser`, the
  `CASE WHEN excluded.name != ''` clause).
- **One-time codes:** keep the existing single-use / short-TTL semantics of delegate codes;
  this change only enriches the payload, not the lifecycle.
