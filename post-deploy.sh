#!/bin/bash

# Update AWS Route53 DNS records with the current IP addresses of the fly app

# Ensure RECORD_NAME and AWS_HOSTED_ZONE_ID are set
if [ -z "$RECORD_NAME" ] || [ -z "$AWS_HOSTED_ZONE_ID" ]; then
    echo "Error: RECORD_NAME and AWS_HOSTED_ZONE_ID must be set."
    exit 1
fi

# Get the list of IPs
IPS_LIST=$(flyctl ips list)
if [ $? -ne 0 ]; then
    echo "Failed to retrieve IP addresses from flyctl."
    exit 1
fi

# Extract the IPv4 address
IPV4=$(echo "$IPS_LIST" | awk '/v4/{print $2}')
if [ -z "$IPV4" ]; then
    echo "No IPv4 address found."
    exit 1
fi

echo "Updating DNS records for $RECORD_NAME with IPv4: $IPV4"

# Update Route53 DNS record
aws route53 change-resource-record-sets \
      --hosted-zone-id "$AWS_HOSTED_ZONE_ID" \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"A\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV4\"}]}}]}"
if [ $? -ne 0 ]; then
    echo "Failed to update DNS records."
    exit 1
fi

echo "DNS update successful."

# Issue TLS certificate if not already issued
echo "Issuing TLS certificate for $RECORD_NAME"
flyctl certs create "$RECORD_NAME" || true # Ignore error if cert already exists

echo "TLS certificate issuance process completed."
