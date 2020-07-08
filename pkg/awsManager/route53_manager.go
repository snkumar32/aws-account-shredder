package awsManager

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	clientpkg "github.com/openshift/aws-account-shredder/pkg/aws"
)

func ListRoute53ForDeletion(client clientpkg.Client) []*string {

	var route53InstancesToBedeleted []*string
	route53HostedZoneOuput, err := client.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		fmt.Println("ERROR:")
		return nil
	} else {
		fmt.Println("ROTE53:")
		fmt.Println(route53HostedZoneOuput)
	}
	for _, hostedZone := range route53HostedZoneOuput.HostedZones {
		route53InstancesToBedeleted = append(route53InstancesToBedeleted, hostedZone.Id)

	}
	return route53InstancesToBedeleted
}

func ClearRoute53Resources(client clientpkg.Client) {
	route53InstancesToBedeleted := ListRoute53ForDeletion(client)
	fmt.Println("ROUTE 53 to be deleted")
	fmt.Print(route53InstancesToBedeleted)
}

func CleanUpAwsRoute53(client clientpkg.Client) error {

	var nextZoneMarker *string

	// Paginate through hosted zones
	for {
		// Get list of hosted zones by page
		hostedZonesOutput, err := client.ListHostedZones(&route53.ListHostedZonesInput{Marker: nextZoneMarker})
		if err != nil {
			fmt.Println("ERROR: ", err)
			return err
		}

		for _, zone := range hostedZonesOutput.HostedZones {

			// List and delete all Record Sets for the current zone
			var nextRecordName *string
			// Pagination again!!!!!
			for {
				recordSet, listRecordsError := client.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{HostedZoneId: zone.Id, StartRecordName: nextRecordName})
				if listRecordsError != nil {
					fmt.Println("Failed to list Record sets for hosted zone %s", *zone.Name)
					return listRecordsError
				}

				changeBatch := &route53.ChangeBatch{}
				for _, record := range recordSet.ResourceRecordSets {
					// Build ChangeBatch
					// https://docs.aws.amazon.com/sdk-for-go/api/service/route53/#ChangeBatch
					//https://docs.aws.amazon.com/sdk-for-go/api/service/route53/#Change
					if *record.Type != "NS" && *record.Type != "SOA" {
						changeBatch.Changes = append(changeBatch.Changes, &route53.Change{
							Action:            aws.String("DELETE"),
							ResourceRecordSet: record,
						})
					}
				}

				if changeBatch.Changes != nil {
					_, changeErr := client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{HostedZoneId: zone.Id, ChangeBatch: changeBatch})
					if changeErr != nil {
						fmt.Println("Failed to delete record sets for hosted zone %s", *zone.Name)
						return changeErr
					}
				}

				if *recordSet.IsTruncated {
					nextRecordName = recordSet.NextRecordName
				} else {
					break
				}

			}

			_, deleteError := client.DeleteHostedZone(&route53.DeleteHostedZoneInput{Id: zone.Id})
			if deleteError != nil {
				fmt.Println("ERROR:", err)
				return deleteError
			}
		}

		if *hostedZonesOutput.IsTruncated {
			nextZoneMarker = hostedZonesOutput.Marker
		} else {
			break
		}
	}

	fmt.Println("Route53 cleanup finished successfully")
	return nil
}
