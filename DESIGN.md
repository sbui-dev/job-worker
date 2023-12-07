---
authors: Steven Bui
state: draft
---

# Remote Job Worker

Remote Job Worker consists of three parts: the job library, the client, and the server. These three components will allow a user to remotely issue a Linux command to be ran on a server.

## Job Library
The job library is responsible for executing Linux commands (i.e. `ls`) via os.exec function calls and also responsible for cpu, memory, and disk io limits via Linux's cgroup.

For this project, there will be no blacklist/whitelist of commands that a client can issue the server to run. Normally, this would be a major security flaw as it would allow anyone who has access to the client to run anything they wish on the server.

### CPU, MEM, DISK IO Limits CGroups

CGroups on linux is a mechanism to limit the amount of CPU, memory, and disk io a process can use.

The limits will use default hardcoded values in the project to save time. Future improvements can introduce a configuration file or command line arguments to override the default values.

Initial CGroup folder will be a `/sys/fs/cgroup/remote-jobs/` folder. Any users will be added under the remote-jobs folder and any job will be under the user folder.

An example file structure is:
```
/sys/fs/cgroup/remote-jobs/
/sys/fs/cgroup/remote-jobs/alice
/sys/fs/cgroup/remote-jobs/alice/job0
/sys/fs/cgroup/remote-jobs/alice/job1
/sys/fs/cgroup/remote-jobs/alice/job2
/sys/fs/cgroup/remote-jobs/bob
/sys/fs/cgroup/remote-jobs/bob/job0
/sys/fs/cgroup/remote-jobs/carl
```

Adding pid to cgroup `echo pid > /sys/fs/cgroup/remote-tasks/alice/cgroup.procs`


## Client/Server

### How to Run

### Proto Specification

Include any `.proto` changes or additions that are necessary for your design.

### Security
#### mTLS
Transport Layer Security (TLS) is a method of authenticating and establishing a secure communication channel between a client and server. As part of TLS, the client verifies the server through a trusted 3rd party known as a Certificate Authority that issued the public/private certificates. After the server verificaiton is completed, they both agree upon an encryption cipher to use for communication. Then the server authenticates the client through basic authentication or some other method.

Mutual TLS (mTLS) is an extension of TLS where both the client and server authenticate themselves through their certificates. Since both client and server certificates come from the same organization CA cert, they trust each other.

##### Setup
Mutual TLS will be used between the client and server to authenticate and encrypt the communication between the two. 

The certificate authory, client, and server certificates will be pregenerated self-signed certificates that will be checked into the code repository for ease of use. This is a bad security practice, but doing it the secure way would require setting up a secret management and/or a certificate provision tool which is out of scope for this project.

Both client and server will stem from the same self-signed CA, which both the client and server will load the CA cert along with their respective public and private keys during startup.

The subject string of the certificate will be:
```
subj /C=US/ST=CA/L=SF/O=JobWorker/CN=Alice/
```
The CN (Common Name) will be used to identify the client. For this project, there will be 3 clients: Alice, Bob, and Carl and one server: Worker.


#### Authorization

### Client UX


### Server UX


## Test Plan

Unit tests will be written for the job library.

Integration testing will be done via the client and server.

A 3rd party program "stress" will be used to stress the machine for CPU, Mem, and Disk IO.