# Fitbit data slurper
GO script to fetch data from fitbit and upload it to an influx database

## Features
Heartrate data

## Requirements
* [GO](https://golang.org)
* [InfluxDB database](https://www.influxdata.com/products/influxdb-cloud/)

## Running
1. Copy the `vars.env.template` file to `vars.env` and fill in the values
2. Run 
```bash
go build
```
 or 
 ```bash
 go install
 ```
3. Run 
```bash
./fitbit
```
