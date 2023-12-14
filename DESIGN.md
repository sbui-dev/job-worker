---
authors: Steven Bui
state: draft
---

# Job Worker

Job Worker consists of three parts: the job library, the grpc client, and the grpc server. These three components together will allow a user to remotely issue Linux commands to be ran on a GRPC server.

For this project, there will be no blacklist/whitelist of commands that a client can issue the server to run. Normally, this would be a major security flaw as it would allow anyone who has access to the client to run anything they wish, which can end up destorying the server.

## Job Library
The job library is responsible for executing Linux commands (i.e. `ls`) via `exec.CommandContext` function calls and also is responsible for cpu, memory, and disk io limits via Linux's cgroup. All output will be stored in a channel for streaming and a buffer in memory in order for concurrent access to a process's output.

The library will use UUIDs as job id to keep track of jobs.

### Library Interface

The following will be exported funcs to be used by the server

```
// NewJob creates a new job
func NewJob(command []string) (*Job, error)

// Start starts a job by executing the command
func (jw *JobInfo) Start()

// Stop stops a job using the id sent in
func (jw *JobInfo) Stop()

// Query shows job status
func (jw *JobInfo) Query(jobID string)

// GetOutputChannel gets job output channel for streaming process output
// return: go channel of output
func (jw *JobInfo) GetOutputChannel() chan string
```

### Data Structures
The following data structures will be used to represent the job/process
```
// key: username
// value: array of job ids
UserJobs := map[string][]string
```

The map will be keeping track and used to look up what jobs a specific user has ran. This provides an added security where it prevent users from accessing other user processes.

```
// jobInfo represents the job
// cancelJob: context to cancel the job
// status: job is either running or stopped
// output: output of the job
type JobInfo struct {
	JobID      string
	status     string
	cancelJob  context.CancelFunc
	outputChan chan string
	command    []string
  output     strings.Builder
}
``````

This jobInfo map with JobInfo struct will be for looking up the process and information related to the process.

```
type JobWorker struct {
    userJobs map[string]JobInfo
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

Start func checks if username exists in `userJobs` map and will handle creation or update a user's job array accordingly. The user command line will be a string array, which the server will use `exec.CommandContext(ctx, []user_command_array)`. Then update the `jobInfo` map with a new job struct containing: a uuid for the job, running status, and the output. The pid will be added to the user's cgroup i.e. `/sys/fs/cgroup/remote-tasks/alice/cgroup.procs`.

Stop func will use the stored context cancel with the `exec.CommandContext()` to kill the process and update the `jobInfo` status to stopped.

Query func will look up the job id inside the `userJob` map first before looking inside the `jobInfo` map for the `job`. Then it display the job status.

GetOutputChannel func is used to get a stream output from the running process.

## Client/Server
### Client
For ease of use, the client will support only a single server with hardcoded a port number 57533 (random high number port to avoid port conflicts). The default server address will be `localhost` but another server ip and port may be specified via command line argument. These variables may be changed in the future with use of command line args, configuration files, or environment variables. Also for ease of use, the client will have 3 user profiles to switch between the pregenerated certs.

#### Client CLI
```
usage: jobclient [<flags>] <command> [<args> ...]

A command-line job client.

Flags:
  --[no-]help               Show context-sensitive help (also try --help-long and --help-man).
  --addr="localhost:50005"  The address to connect to
  --user="alice"            Name of user: alice, bob, carl

Commands:
help [<command>...]
    Show help.

start <command>...
    Start a job

stop <id>
    Stop a job

query <id>
    Query a job
```

After a job is started, the client will start streaming the output from received from the server until the job has completed.

After a job is queried, the client will start streaming from the beginning to latest. If the job is still running, it will continue to stream the output from the server.

The client will receive a confirmation that a job is stopped

### Server
The server will be responsible for authn via mTLS, authz, and the jobs. It will not have any persistent storage. Therefore, server restarts will wipe out any job information it had stored in memory.

To run the server: `jobserver`

### Proto Specification

```
message WorkerStartRequest {
  repeated string command = 1;
}

message WorkerStopRequest{
  string jobID = 1;
}

message WorkerQueryRequest{
  string jobID = 1;
}

message WorkerStartResponse {
  string jobID = 1;
  string log = 2;
}

message WorkerStopResponse {
}

message WorkerQueryResponse {
  string jobID = 1;
  string log = 2;
}

service Worker {
  rpc JobStop(WorkerStopRequest) returns (WorkerStopResponse) {}

  rpc JobStart(WorkerStartRequest) returns (stream WorkerStartResponse) {}

  rpc JobQuery(WorkerQueryRequest) returns (stream WorkerQueryResponse) {}
}
```

### Streaming
The server will use the oberserver pattern to broadcast output to multiple concurrent clients. It maintain a list of clients that are subscribed to a specific job. Each time the server receives output from the `OutputChan`, it'll replicate the output to the clients via 

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

Jobs are associated with the user that created the job and authorization will be done by checking if a user has access to a job before displaying any information or stopping the job. The username will come from the TLS certificate's common name field. The affected APIs are StopJob and QueryJob that require a job id. What this means is the user Alice will not be able to access user Bob's jobs even if she has the job id.

Security-wise, it's okay since the TLS certificates are signed by a CA cert and the server trusts the CA.

## Build / Package

A simple `build.sh` script will be provided to build the client and server with their pregenerated certificates. The end result will be a `bin` folder containing the binaries and the certificates.

In the future, github actions can be used to build and package the client and server. The end result will be a tar.gz file that can be unpacked and ran without having to generate certificates.

## Test Plan

Unit tests will be written to test the job library, authn, and authz.

Integration testing will be done via the client and server to ensure it's functioning correctly.

A 3rd party program "stress" will be used to stress the machine for CPU, Mem, and Disk IO limits.