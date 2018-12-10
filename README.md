# MicroMDM - Abacus
In addition, to official README of the source project: https://github.com/micromdm/micromdm

## Requirements

Make sure, go version 1.11 or newer is installed:
`go version`

For testing, install `goose`
`brew install goose`

## Compiling
Run build in root directory of the project
`make build`

Will generate our executables in 
```
./build/darwin
./build/linux
```

Will also generate a `docker-compose-dev.yaml` file, for further configuration.

Choose your system (darwin/linux) and go to the directory.

```
# Executable for configuration and setup
./build/(darwin|linux)/mdmctl

# Executable for running the server
./build/(darwin|linux)/micromdm
```

## Testing
### Mysql - Only for Test environment!
#### Setup test account:
* username: micromdm
* password: micromdm
```
mysql -u root
mysql> CREATE USER 'micromdm'@'localhost' IDENTIFIED BY 'micromdm';
mysql> GRANT ALL PRIVILEGES ON *.* TO 'micromdm'@'localhost' IDENTIFIED BY 'micromdm';
```
#### Run tests
```
make db-mysql-migrate-test
make db-mysql-test
```

## Setup
### mdmctl 
#### Configure later MDM Service
```
./mdmctl config set \
    -api-token secret \
    -name mdmexample \
    -server-url https://mdm.abacus.ch/
./mdmctl config switch -name mdmexample
```

#### Assign Apple DEP Token
1. Generate public key
```
./mdmctl get dep-tokens \
    -export-public-key ../../assets/mdm-certificates/dep_public_key.pem
```
2. Get DEP Token from business.apple.com
   1. Upload File from `./assets/mdm-certificates/dep_public_key.pem` to https://business.apple.com
   2. Download now the p7m file, which we will import to the mdm server, save as `./assets/mdm-certificates/dep_token.p7m`
   3. Set DEP Token for server:
   ```
   ./mdmctl apply dep-tokens \
       -import ../../assets/mdm-certificates/dep_token.p7m
   ```
3. Make sure, import worked --> result returned for
```
./mdmctl get dep-tokens
```

#### Assign Apple Push Certificate (APNS Cert)
To assign an Apple Push Certificate, start the server first (no Mysql database connection required, we won't store the certificate in the Mysql database, but locally in a document store.)
We will need two Terminals/Consoles.

Run following command in Terminal 1
```
sudo ./micromdm serve \
    -config-path $(echo $(pwd)/../../build/) \
    -api-key secret \
    -tls-cert ./fullchain.pem \
    -tls-key ./privkey.pem \
    -server-url https://mdm.abacus.ch/
```

Now, when the server is running, add the Push certificate, from Terminal 2.
```
./mdmctl mdmcert upload \
    -password secret \
    -cert ../../assets/mdm-certificates/MDM_Abacus_Research_AG_Certificate.pem \
    -private-key ../../assets/mdm-certificates/PushCertificatePrivateKey.key
```

### micromdm locally
After configuring the MDM Service, run it.

```
sudo ./micromdm serve \
    -config-path $(echo $(pwd)/../../assets/) \
    -api-key secret \
    -tls-cert ./fullchain.pem \
    -tls-key ./privkey.pem \
    -server-url https://mdm.abacus.ch/ \
    -command-webhook-url http://127.0.0.1:5000/webhook \
    -mysql-username micromdm \
    -mysql-password micromdm \
    -mysql-database micromdm_test \
    -mysql-host 127.0.0.1 \
    -mysql-port 3306
```

### BoltDB - Document Store
Currently, some data is still being stored in the Mysql independant document store.
If you run this MDM in OpenShift, this file needs to be provided as "secret" in OpenShift.
`/assets/micromdm.db`

By using the Bolter
`bolter -f ./assets/micromdm.db`

#### Certificates
mdm.ServerConfig

#### DEP Token from Apple (.p7m)
mdm.DEPToken

#### scep_certificates
scep_certificates

## Operations
### Build with Docker File
Use ./Dockerfile to run the docker build
```
docker build . -t micromdm
```
### Run
Provide the path to the micromdm.db as variable `/data`
Due to our reverse proxy, we won't provide
* tls-cert
* tls-key

```
docker run -v /absolute/path/to/micromdm/assets/:/data  micromdm \
    micromdm serve \
    -config-path /data \
    -api-key secret \
    -server-url https://mdm.abacus.ch/ \
    -command-webhook-url http://127.0.0.1:5000/webhook \
    -mysql-username micromdm \
    -mysql-password micromdm \
    -mysql-database micromdm_test \
    -mysql-host 127.0.0.1 \
    -mysql-port 3306
```
