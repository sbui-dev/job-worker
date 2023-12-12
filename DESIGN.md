---
authors: Steven Bui
state: draft
---

# Job Worker

Job Worker consists of three parts: the job library, the grpc client, and the grpc server. These three components together will allow a user to remotely issue Linux commands to be ran on a GRPC server.

For this project, there will be no blacklist/whitelist of commands that a client can issue the server to run. Normally, this would be a major security flaw as it would allow anyone who has access to the client to run anything they wish, which can end up destorying the server.

## Job Library
The job library is responsible for executing Linux commands (i.e. `ls`) via `exec.Command` function calls and also is responsible for cpu, memory, and disk io limits via Linux's cgroup. All output will be stored in a buffer in memory in order for concurrent access to a process's output.

The library will use UUIDs as job id to keep track of jobs.

### Library Interface

The following will be exported funcs to be used by the server

```
// StartJob starts a job by executing the command
// input: the name of the user and command to run
// output: response containing job output and id; error if any
(j *JobWorker) StartJob(username, command string) (response, error)

// StopJob stops a job using the id sent in
// input: id of job
// output: error if any; nil respresents success
(j *JobWorker) StopJob(id string) error

// QueryJob shows job status and stream output
// input: id of job
// output: response containing job output and id; error if any
(j *JobWorker) QueryJob(id string) (response, error)
```

### Data Structures
The following data structures will be used to represent the job/process
```
// key: username
// value: array of job ids
userJobs := map[string][]string
```

The map will be keeping track and used to look up what jobs a specific user has ran. This provides an added security where it prevent users from accessing other user processes.

```
// processInfo represents the process
// pid: pid assigned by the linux os
// status: process is either running or stopped
// output: output of the process
type processInfo struct {
    pid string
    status string
    output strings.Builder
}

// key: job id
// value: pid info
jobInfo := map[string]processInfo
``````

This jobInfo map with processInfo struct will be for looking up the process and information related to the process.

```
type JobWorker struct {
    userJobs map[string][]string
    jobInfo map[string]processInfo
}
```

The library will be initialized with JobWorker struct. Since the library supports concurrent jobs and users, mutexes will be used to ensure consistency for the data structures and files being modified.

### CPU, MEM, DISK IO Limits via CGroup

CGroups on linux is a mechanism to limit the amount of CPU, memory, and disk io a process can use. There are two versions of cgroups which are v1 and v2. Modern Linux OSes will use cgroup v2, which this library will only support.

The limits will use default hardcoded values in the project to save time. Future improvements can introduce a configuration file or command line arguments to override the default values.

Initial cgroup folder will be a `/sys/fs/cgroup/jobworker/` folder. Any users will be added under the jobworker folder and any job will be added in the `cgroup.procs` file.

An example file structure is:
```
/sys/fs/cgroup/jobworker/
/sys/fs/cgroup/jobworker/alice
/sys/fs/cgroup/jobworker/bob
/sys/fs/cgroup/jobworker/carl
```

The following files will be edited: `cpu.max`, `mem.max`, and `io.max` with the contents with the hardcoded default values 100000 microseconds CPU time in a 200000 microsecond period, 134217728 (128MB) for memory max, 1MB for wbps and 120 wiops:

**cpu.max**<br>
`100000 200000`

**mem.max**<br>
`134217728`

**io.max**<br>
`8:0 rbps=max wbps=1048576 riops=max wiops=120`

`8:0` represents the `/dev/sda`

New jobs will add their pid to cgroup file in their respective user folders. For example: `/sys/fs/cgroup/jobworker/alice/cgroup.procs`

### Job Life Cycle

StartJob func checks if username exists in `userJobs` and will handle creation or update a user's job array accordingly. It will split the user command into a string array and use `exec.Command([]user_command_array)`. Then update the `jobInfo` map with a new job struct containing the pid, running status, and the output. The pid will be added to the user's cgroup i.e. `/sys/fs/cgroup/remote-tasks/alice/cgroup.procs`.

StopJob func use `exec.Command("kill -9 <pid>")` and update the `pidInfo` status to stopped.

QueryJob func will look up the pid inside the `userJob` map first before looking inside the `jobInfo` map for the `pidInfo`. Then it would stream the output.

## Client/Server
### Client
For ease of use, the client will support only a single server with hardcoded a port number 57533 (random high number port to avoid port conflicts). The default server address will be `localhost` but another server ip may be specified via command line argument. These variables may be changed in the future with use of command line args, configuration files, or environment variables. Also for ease of use, the client will have 3 user profiles to switch between the pregenerated certs.

#### Client CLI
To run the client: `jobworker [flags] [start/stop/query] command/pid`

| command line option | description |
| ----------- | ----------- |
| -u | changes user profile: alice, bob, carl |
| -h | changes the host |

Examples:
```
jobworker -u bob start ls
jobworker -h 127.0.0.1 stop 3
jobworker query 3
```

After a job is started, the client will periodically check the stream to display new output from the job until it has completed.

After a job is queried, the client will output everything from the beginning to latest. If the job is still running, it will periodically check the stream to display new output from the job.

The client will receive a confirmation that a job is stopped

### Server
The server will be responsible for authn via mTLS, authz, and the jobs. It will not have any persistent storage. Therefore, server restarts will wipe out any job information it had stored in memory.

To run the server: `jobserver`

### Proto Specification

```
message WorkerStartRequest {
  string command = 1;
}

message WorkerPIDRequest{
  string pid = 1;
}

message WorkerResponse {
  string pid = 1;
  string log = 2;
}

service Worker {
  rpc JobStop(WorkerPIDRequest) returns (WorkerResponse) {}

  rpc JobStart(WorkerStartRequest) returns (stream WorkerResponse) {}

  rpc JobQuery(WorkerPIDRequest) returns (stream WorkerResponse) {}
}

```

### Security
#### mTLS
Transport Layer Security (TLS) is a method of authenticating and establishing a secure communication channel between a client and server. As part of TLS, the client verifies the server through a trusted 3rd party known as a Certificate Authority that issued the public/private certificates. After the server verificaiton is completed, they both agree upon an encryption cipher to use for communication. Then the server authenticates the client through basic authentication or some other method.

Mutual TLS (mTLS) is an extension of TLS where both the client and server authenticate themselves through their certificates. Since both client and server trust each other's CA certificates, they trust each other.

By default, grpc uses either TLS 1.2 or TLS 1.3. The TLS 1.2 is the current standard but is full of security vulnerable ciphers. The industry is slowly moving toward TLS 1.3 for better security and performance. The server will be set to only support TLS 1.3 as a minimum via the `tls.Config`. In addition, only the modern ciphers will be in the cipher pool to be used by specifying them in the cert pool.

List of modern ciphers:
https://developers.cloudflare.com/ssl/reference/cipher-suites/recommendations/

##### Setup
Mutual TLS will be used between the client and server to authenticate and encrypt the communication between the two. 

The certificate authory, client, and server certificates will be pregenerated self-signed certificates that will be checked into the code repository for ease of use. This is a bad security practice, but doing it the secure way would require setting up a secret management and/or a certificate provision tool which is out of scope for this project.

The client certificates from the same self-signed CA and the server will have it's own self-signed CA cert. Both the client and server will load the other's CA cert along with their respective public and private keys during startup.

Certificates will be RSA 4096 bits with SHA256 hash. OpenSSL recommends using at least 2048 bits for secured certificates. There is a trade off between 2048 and 4096 bits which increased server CPU usage for security. However, this particular server isn't expected to have a high load of traffic so it'll be using the more secure 4096 bits.

Alternatively in the future, ECDSA certificates can be used to increase security and performance. ECC with shorter keys provides same amount of security as RSA. RSA is the current standard that has been used for a long time while is the modern ECDSA is gaining adoption. The certificates will be RSA since it is easier to setup and use.

Certificates will be generated on command line using openssl. Example:
```
openssl req -x509                                        \
  -newkey rsa:4096                                       \
  -nodes                                                 \
  -days 365                                              \
  -keyout server_ca_key.pem                              \
  -out server_ca_cert.pem                                \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-server_ca/   \
  -sha256

openssl genrsa -out server_key.pem 4096
openssl req -new                                      \
  -key server_key.pem                                 \
  -days 365                                           \
  -out server_csr.pem                                 \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-server/

openssl x509 -req           \
  -in server_csr.pem        \
  -CAkey server_ca_key.pem  \
  -CA server_ca_cert.pem    \
  -days 365                 \
  -out server_cert.pem      \
  -set_serial 1             \
  -sha256
```

The subject string of the client certificate will be:
```
subj /C=US/ST=CA/L=SF/O=JobWorker/CN=alice/
```
The CN (Common Name) will be used to identify the client. For this project, there will be 3 clients: alice, bob, and carl and one server: worker.

#### Authorization

GRPC authorization will be done with Authz golang package with a simple authorization requiring username, which is the TLS certificate common name. Static unary and stream interceptors will be created to perform the authz check. The server will verify if the username matches one of the known users that has access. If the username field matches the hardcoded "alice" or "bob" values, full access will be allowed to the APIs by adding the following key-value pairs in the metadata: 

| API         | Metadata Key Pairs |
| ----------- | ----------- |
| JobStart    | allow_start=true |
| JobStop     | allow_stop=true |
| JobQuery    | allow_query=true |

Alice and Bob will have full access. Carl will have a valid certificate but will not have any access to any API and can be used to test authorization.

This method is very similar to passing an API key. Security-wise, it's okay since the TLS certificates are signed by a CA cert and the server trusts the CA. It's not a scalable solution because it's static. Adding a user database resolves the static user issue but performance remains an issue for each API request to check against the database. Extending the implementation to a token auth would resolve the scalability issue.

## Build / Package
Github Actions will be used to build and package the client and server. The end result will be a tar.gz file that can be unpacked and ran without having to generate certificates.

## Test Plan

Unit tests will be written to test the job library, authn, and authz.

Integration testing will be done via the client and server to ensure it's functioning correctly.

A 3rd party program "stress" will be used to stress the machine for CPU, Mem, and Disk IO limits.