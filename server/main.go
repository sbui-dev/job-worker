package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	worker "github.com/sbui-dev/jobworker/data/proto"
	joblib "github.com/sbui-dev/jobworker/lib"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	certFolder     = "../data/certs/"
	clientCAPath   = "client_ca_cert.pem"
	serverCertPath = "server_cert.pem"
	serverKeyPath  = "server_key.pem"
)

var port = flag.Int("port", 50005, "the port to serve on")

type workerServer struct {
	JobWorker      *joblib.JobWorker
	mutex          sync.Mutex
	jobSubscribers map[*joblib.JobInfo][]chan string
	worker.UnimplementedWorkerServer
}

func newWorkerServer() *workerServer {
	jw := joblib.NewJobWorker()
	js := make(map[*joblib.JobInfo][]chan string)
	return &workerServer{JobWorker: jw, jobSubscribers: js}
}

func getUserFromCertificate(ctx context.Context) (string, error) {
	fmt.Println("getting user name")
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no peer found")
	}

	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "unexpected peer transport credentials")
	}

	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		return "", status.Error(codes.Unauthenticated, "could not verify peer certificate")
	}

	username := tlsAuth.State.VerifiedChains[0][0].Subject.CommonName
	fmt.Printf("user name is %s\n", username)
	return username, nil
}

func (w *workerServer) JobStart(ctx context.Context, req *worker.WorkerStartRequest) (*worker.WorkerStartResponse, error) {
	fmt.Printf("Creating new job: %s\n", req.Command)

	username, err := getUserFromCertificate(ctx)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, err
	}

	newJob, err := joblib.NewJob(req.Command)
	if err != nil {
		log.Println(err.Error())
	}

	// add to array
	w.JobWorker.AddJob(username, newJob)

	fmt.Println("Job starting")
	go newJob.Start()

	// outChan := newJob.GetOutputChannel()

	// for {
	// 	select {
	// 	case <-stream.Context().Done():
	// 		log.Printf("user closed connection")
	// 		return nil
	// 	case out, ok := <-outChan:
	// 		if !ok {
	// 			log.Printf("channel closed")
	// 			return nil
	// 		}
	// 		//fmt.Println("sending response")
	// 		err = stream.Send(&worker.WorkerStartResponse{JobId: newJob.JobID, Log: out})
	// 		if err != nil {
	// 			fmt.Println(err.Error())
	// 			return err
	// 		}
	// 	}
	// }

	return &worker.WorkerStartResponse{JobId: newJob.JobID}, nil
}

func (w *workerServer) JobStop(ctx context.Context, req *worker.WorkerStopRequest) (*worker.WorkerStopResponse, error) {
	log.Printf("Stop job: %s\n", req.JobId)
	username, err := getUserFromCertificate(ctx)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, err
	}

	myJob, err := w.JobWorker.FindJob(username, req.JobId)
	if err != nil {
		return nil, err
	}

	if myJob.IsRunning() {
		myJob.Stop()
	}

	return &worker.WorkerStopResponse{}, nil

}

func (w *workerServer) JobQuery(ctx context.Context, req *worker.WorkerQueryRequest) (*worker.WorkerQueryResponse, error) {
	fmt.Printf("Query job: %s\n", req.JobId)
	username, err := getUserFromCertificate(ctx)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, err
	}

	myJob, err := w.JobWorker.FindJob(username, req.JobId)
	if err != nil {
		return nil, err
	}

	fmt.Printf("job status is %s", myJob.Status())

	return &worker.WorkerQueryResponse{Status: myJob.Status()}, nil

}

func (w *workerServer) SubscribeToJob(job *joblib.JobInfo) (<-chan string, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	newChan := make(chan string)
	streamArray, ok := w.jobSubscribers[job]
	if !ok {
		w.jobSubscribers[job] = []chan string{newChan}
		return nil, nil
	}
	streamArray = append(streamArray, newChan)
	w.jobSubscribers[job] = streamArray
	return newChan, nil
}

func (w *workerServer) UnsubscribeToJob(job *joblib.JobInfo) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	// TODO fix
}

func (w *workerServer) Broadcast() {
	for {
		for job, chanArray := range w.jobSubscribers {
			myChan := job.GetOutputChannel()
			select {
			case out, ok := <-myChan:
				if !ok {
					log.Printf("channel closed")
					return
				}
				fmt.Printf("broadcasting %s\n", out)
				fmt.Println(len(chanArray))
				for _, sub := range chanArray {
					sub <- out
					fmt.Println("broadcasted")
				}
			}
		}
	}
}

func (w *workerServer) StreamJobLogs(jobID string, myChan <-chan string, stream worker.Worker_JobLogServer) {
	for {
		select {
		case <-stream.Context().Done():
			log.Printf("user closed connection")
			return
		case out, ok := <-myChan:
			fmt.Println("received data to send")
			if !ok {
				log.Printf("channel closed")
				return
			}

			err := stream.Send(&worker.WorkerLogResponse{JobId: jobID, Log: out})
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	}
}

func (w *workerServer) JobLog(req *worker.WorkerLogRequest, stream worker.Worker_JobLogServer) error {
	fmt.Printf("Log job: %s\n", req.JobId)
	username, err := getUserFromCertificate(stream.Context())
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	myJob, err := w.JobWorker.FindJob(username, req.JobId)
	if err != nil {
		return err
	}

	myChan, err := w.SubscribeToJob(myJob)
	if err != nil {
		return err
	}

	go w.StreamJobLogs(req.JobId, myChan, stream)

	return nil
}

func main() {
	flag.Parse()

	log.Printf("server starting on port %d...\n", *port)

	cert, err := tls.LoadX509KeyPair(certFolder+serverCertPath, certFolder+serverKeyPath)
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}

	ca := x509.NewCertPool()
	caBytes, err := os.ReadFile(certFolder + clientCAPath)
	if err != nil {
		log.Fatalf("failed to read ca cert %q: %v", clientCAPath, err)
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		log.Fatalf("failed to parse %q", clientCAPath)
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    ca,
	}

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

	workerServer := newWorkerServer()
	worker.RegisterWorkerServer(grpcServer, workerServer)
	go workerServer.Broadcast()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
