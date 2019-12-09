## Event Exporter

This tool is used to scraper Kubernetes events and export it to a sink (for example like S3). It effectively runs a watch on the apiserver, detecting as granular as possible all changes to the event objects. Event scrapper exports only to S3.

#### To run it locally

```go run main.go controller.go -apiServerAddr="https://5FBCEAC726E0C6806EAD626A8065B5BF.yl4.us-east-2.eks.amazonaws.com" -kubeconfigPath="/Users/nithmu/.kube/config" ```