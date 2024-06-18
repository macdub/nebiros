package Utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
)

type AksEntry struct {
	Name            string `json:"name"`
	ResourceGroup   string `json:"resourceGroup"`
	ClusterName     string `json:"clusterName"`
	PowerState      string `json:"powerState,omitempty"`
	ClusterResponse armcontainerservice.ManagedClustersClientGetResponse
}

type AksEntryList struct {
	Entries []*AksEntry
}

func NewAksEntryList(path string) (RetList *AksEntryList, err error) {
	RetList = &AksEntryList{}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return nil, err
	}

	if !json.Valid(data) {
		msg := fmt.Sprintf("Input data is invalid json. FilePath: %s\nData: %s\n", path, string(data[:]))
		return nil, errors.New(msg)
	}

	err = json.Unmarshal(data, &RetList.Entries)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return nil, err
	}

	return RetList, nil
}

func (A *AksEntryList) GetNameColumnMaxLength() (Ret int) {
	Ret = 15
	for _, Entry := range A.Entries {
		if len(Entry.Name) > Ret {
			Ret = len(Entry.Name)
		}
	}
	return Ret
}

func (A *AksEntryList) GetResourceGroupColumnMaxLength() (Ret int) {
	Ret = 15
	for _, Entry := range A.Entries {
		if len(Entry.ResourceGroup) > Ret {
			Ret = len(Entry.ResourceGroup)
		}
	}
	return
}

func (A *AksEntryList) GetClusterNameColumnMaxLength() (Ret int) {
	Ret = 15
	for _, Entry := range A.Entries {
		if len(Entry.ClusterName) > Ret {
			Ret = len(Entry.ClusterName)
		}
	}
	return
}

func (A *AksEntryList) PrintConfigTable() string {

	var sb strings.Builder
	nameWidth := A.GetNameColumnMaxLength() + 2
	rgWidth := A.GetResourceGroupColumnMaxLength() + 2
	cnWidth := A.GetClusterNameColumnMaxLength() + 2
	nameSpan := strings.Repeat(H, nameWidth+2)
	rgSpan := strings.Repeat(H, rgWidth+2)
	cnSpan := strings.Repeat(H, cnWidth+2)

	// @TODO Unify this string building
	NameFmt := "%-" + fmt.Sprintf("%ds", nameWidth)
	RGFmt := "%-" + fmt.Sprintf("%ds", rgWidth)
	CNFmt := "%-" + fmt.Sprintf("%ds", cnWidth)
	RowFormat := V + " " + NameFmt + " " + V + " " + RGFmt + " " + V + " " + CNFmt + " " + V + "\n"

	// Header
	sb.WriteString(TL + nameSpan + HD + rgSpan + HD + cnSpan + TR + "\n")
	sb.WriteString(fmt.Sprintf(RowFormat, "Name", "Resource Group", "Cluster Name"))
	sb.WriteString(VR + nameSpan + X + rgSpan + X + cnSpan + VL + "\n")

	// Body
	for _, Entry := range A.Entries {
		sb.WriteString(fmt.Sprintf(RowFormat, Entry.Name, Entry.ResourceGroup, Entry.ClusterName))
	}

	// footer
	sb.WriteString(BL + nameSpan + HU + rgSpan + HU + cnSpan + BR + "\n")
	return sb.String()
}
