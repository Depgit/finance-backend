# Project finance-manage

One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```


Added auth (models, password hashing, JWT helpers).
Added middleware JWT + role middleware.
Added handlers auth routes: /auth/register, /auth/login, /api/me, /admin/approve/:id.
Hooked migration into internal/server.NewServer.
Added CLI user.create-admin to create the one-time ADMIN: run with go run [user.create-admin](http://_vscodecontentref_/9) -email admin@x -password secret.
Exposed DB accessor in database to support auth package.
Built the project and fetched dependencies.




Added .env at project root with PORT, JWT_SECRET, ADMIN_EMAIL, and SMTP placeholders.
Added email.go — simple SMTP notifier that no-ops if SMTP_HOST is not set.
Wired notifier into registration (auth_handlers.go) to email ADMIN_EMAIL asynchronously after a successful signup.
Built the project to confirm compilation.


Backend: added GET /admin/pending to list unapproved users (admin-only).
Frontend:
api.ts — API helpers and token storage.
Login.tsx, Register.tsx, Admin.tsx — pages for auth and admin approval.
Reworked App.tsx to use react-router-dom with a simple Nav and Home.
Styling in index.css for a clean look.
Added react-router-dom to package.json.
