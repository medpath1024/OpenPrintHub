package printer

import "testing"

func TestDetectCapabilitiesForSingaporeLabelPrinters(t *testing.T) {
	tests := []struct {
		name                  string
		printerName           string
		wantClass             string
		wantLanguage          string
		wantJobType           string
		wantPrimaryCommandSet string
	}{
		{
			name:                  "current Xprinter XP-420B uses TSPL raw",
			printerName:           "Xprinter XP-420B",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageTSPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetTSPL,
		},
		{
			name:                  "TSC desktop family uses TSPL raw",
			printerName:           "TSC TE200",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageTSPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetTSPL,
		},
		{
			name:                  "Zebra desktop family uses ZPL raw",
			printerName:           "Zebra ZD421",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageZPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetZPL,
		},
		{
			name:                  "BIXOLON label printer uses ZPL emulation raw",
			printerName:           "BIXOLON XD3-40d",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageZPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetZPL,
		},
		{
			name:                  "GoDEX label printer uses GZPL raw path",
			printerName:           "GoDEX GE300",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageZPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetZPL,
		},
		{
			name:                  "Honeywell desktop label printer uses ZSim raw path",
			printerName:           "Honeywell PC42t Plus",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageZPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetZPL,
		},
		{
			name:                  "Brother TD label printer uses ZPL emulation raw path",
			printerName:           "Brother TD-4420DN",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageZPL,
			wantJobType:           JobTypeRawValue,
			wantPrimaryCommandSet: CommandSetZPL,
		},
		{
			name:                  "Brother QL stays image based",
			printerName:           "Brother QL-820NWB",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageImage,
			wantJobType:           JobTypeImageValue,
			wantPrimaryCommandSet: CommandSetImage,
		},
		{
			name:                  "DYMO stays image based",
			printerName:           "DYMO LabelWriter 450 Turbo",
			wantClass:             DeviceClassLabel,
			wantLanguage:          LabelLanguageImage,
			wantJobType:           JobTypeImageValue,
			wantPrimaryCommandSet: CommandSetImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCapabilities("", tt.printerName, "")
			if got.DeviceClass != tt.wantClass {
				t.Fatalf("DeviceClass = %q, want %q", got.DeviceClass, tt.wantClass)
			}
			if got.PreferredLabelLanguage != tt.wantLanguage {
				t.Fatalf("PreferredLabelLanguage = %q, want %q", got.PreferredLabelLanguage, tt.wantLanguage)
			}
			if got.PreferredJobType != tt.wantJobType {
				t.Fatalf("PreferredJobType = %q, want %q", got.PreferredJobType, tt.wantJobType)
			}
			if len(got.CommandSets) == 0 || got.CommandSets[0] != tt.wantPrimaryCommandSet {
				t.Fatalf("CommandSets = %#v, want first %q", got.CommandSets, tt.wantPrimaryCommandSet)
			}
		})
	}
}

func TestDetectCapabilitiesExcludesDocumentAndReceiptPrinters(t *testing.T) {
	tests := []struct {
		name        string
		printerName string
		wantClass   string
		wantMedia   string
	}{
		{
			name:        "HP LaserJet is document printer",
			printerName: "HP LaserJet MFP M141w",
			wantClass:   DeviceClassDocument,
			wantMedia:   MediaKindA4Document,
		},
		{
			name:        "Brother MFC is document printer",
			printerName: "Brother MFC L3770CDW series",
			wantClass:   DeviceClassDocument,
			wantMedia:   MediaKindA4Document,
		},
		{
			name:        "Xprinter receipt family is not label",
			printerName: "Xprinter XP-N160",
			wantClass:   DeviceClassReceipt,
			wantMedia:   MediaKindReceiptRoll,
		},
		{
			name:        "BIXOLON receipt family is not label",
			printerName: "BIXOLON SRP-330",
			wantClass:   DeviceClassReceipt,
			wantMedia:   MediaKindReceiptRoll,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCapabilities("", tt.printerName, "")
			if got.DeviceClass != tt.wantClass {
				t.Fatalf("DeviceClass = %q, want %q", got.DeviceClass, tt.wantClass)
			}
			if got.MediaKind != tt.wantMedia {
				t.Fatalf("MediaKind = %q, want %q", got.MediaKind, tt.wantMedia)
			}
			if got.PreferredLabelLanguage != LabelLanguageNone {
				t.Fatalf("PreferredLabelLanguage = %q, want %q", got.PreferredLabelLanguage, LabelLanguageNone)
			}
			if got.PreferredJobType != JobTypeNone {
				t.Fatalf("PreferredJobType = %q, want %q", got.PreferredJobType, JobTypeNone)
			}
		})
	}
}

func TestApplyCapabilitiesEnrichesPrinterInfo(t *testing.T) {
	info := ApplyCapabilities(PrinterInfo{
		ID:   "Xprinter_XP-420B",
		Name: "Xprinter XP-420B",
	})

	if info.DeviceClass != DeviceClassLabel {
		t.Fatalf("DeviceClass = %q, want %q", info.DeviceClass, DeviceClassLabel)
	}
	if info.PreferredLabelLanguage != LabelLanguageTSPL {
		t.Fatalf("PreferredLabelLanguage = %q, want %q", info.PreferredLabelLanguage, LabelLanguageTSPL)
	}
	if info.DPI != 203 {
		t.Fatalf("DPI = %d, want 203", info.DPI)
	}
	if info.CapabilitySource == "" {
		t.Fatal("CapabilitySource is empty")
	}
}
