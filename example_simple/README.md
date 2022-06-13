## Steps for Local
```shell
git clone https://github.com/atekoa/dvc-http-remote.git
cd dvc-http-remote
docker-compose up -d --build --force-recreate remote

cd example_simple/
git init
dvc init
dvc remote add -d localhost http://localhost:8080/remote?remote=0
dvc remote modify localhost ssl_verify false
dvc add dvc-logo.png
dvc push -vv
```

## Steps for Azure
```shell
git clone https://github.com/atekoa/dvc-http-remote.git
cd dvc-http-remote
docker-compose up -d --build --force-recreate remote
```

> Remember to set variables AZURE_STORAGE_URL/AZURE_CONNECTION_STRING in the server

```shell
cd example_simple/
git init
dvc init
dvc remote add -d localhost http://localhost:8080/remote?remote=1
dvc remote modify localhost ssl_verify false
dvc add dvc-logo.png
dvc push -vv
```
