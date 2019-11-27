## Event Scrapper

This tool is used to scraper Kubernetes events and export it to a sink (for example like S3). It effectively runs a watch on the apiserver, detecting as granular as possible all changes to the event objects. Event scrapper exports only to S3.
