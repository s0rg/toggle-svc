# toggle-svc

 Feature-toggler MVP

 It allows you to manage feature-toggles for your applications, and switch them on/off/or for some amount of clients.

# Set-up

- `git clone https://github.com/s0rg/toggle-svc.git`
- `cd toggle-svc`
- `make docker-build`
- `docker-compose up`

# Usage

Create some apps, they acts as namespaces for your features.

`curl -d '{
    "apps": ["ios", "web", "android"]
}' http://localhost:8080/apps/add`

Add some keys for "web"-application, note the `key3` has been initially disabled.

`curl -d '{
    "app": "web",
    "version": "1.0",
    "platforms": ["ie6"],
    "keys": [
        {"name": "key1", "enabled": true},
        {"name": "key2", "enabled": true},
        {"name": "key3", "enabled": false}
]}' http://localhost:8080/toggles/add`


Edit key3 for web to cover only 50% cients.

`curl -d '{
   "app": "web",
   "version": "1.0",
   "platform": "ie6",
   "key": "key3",
   "rate": 0.5
}' http://localhost:8080/toggles/edit`

Get some toggles.

`curl -d '{
    "app": "web",
    "version": "1.0",
    "platform": "ie6"
}' http://localhost:8080/client/code-toggles`

Retrive toggles (without any counters increase).

`curl -H "X-CodeToggleID: your-toggle-id" -d '{
    "app": "web",
    "version": "1.0",
    "platform": "ie6"
}' http://localhost:8080/client/code-toggles`

Updates client alive ttl.

`curl -d '{"id": "your-toggle-id"}' http://localhost:8080/client/alive`

