# hive-fleet [![Go Report Card](https://goreportcard.com/badge/github.com/clglavan/hive-fleet)](https://goreportcard.com/report/github.com/clglavan/hive-fleet)

 `hive-fleet` is a distributed, scalable load-testing tool, built on top of https://github.com/codesenberg/bombardier (v1.2.5).

It leverages the power of `Google Cloud Functions` to scale bombardier to any possible range. Google and budget are your only limits.

# Take it for a spin

    git clone https://github.com/clglavan/hive-fleet.git

Prerequisites:
- a billing enabled GCP project
- a service account for it, with exported credentials
- installed and configured gcloud where you run hive-fleet
- asume that any run of this program could result in costs in GCP

setup your config.yml file and pass it as a command line parameter

    clients: 5     => number of CF you want to start
    local: 0       => used for debugging, leave 0 when load-testing
    debug: 0       => a slightly more verbose setting for local
    deploy_function: 1      => leave to yes at least the first time
    function_memory: 256    => function memory
    function_timeout: 120   => function timeout
    function_region: "us-central1"       => CF region
    credentials: "{path-to-my-credentials}.json" => Path to your google credentials
    concurrency: 1  => how many conccurent threads each CF uses 
    number: 10      => the number of requests each CF sends
    url: https://google.com => url to test

Before you proceed, please check Google's pricing for Cloud Functions: https://cloud.google.com/functions/pricing#cloud-functions-pricing

Run the hive-mind with one of the two binaries

    ./hive-mind-darwin-amd64 config.yml
    - or -
    ./hive-mind-linux-amd64 config.yml


In the folder you will see a generated `report.html`

# Local setup

If you want to test it out without involing GCP at all, you can first start a local webserver in 

    ./commander/local/local

and from the config values above, change:

    local: 1

This will instruct hive-fleet to call a local endpoint insted of the cloud function endpoint, and run everything on your local machine

Run the hive-mind with one of the two binaries

    ./hive-mind-darwin-amd64 config.yml
    - or -
    ./hive-mind-linux-amd64 config.yml
