# Printer Capability Catalog

Date: 2026-06-22

This document is the working catalog for OpenPrintHub printer capability detection. It focuses on printers that are visible in the Singapore clinic / retail / logistics market and on printer languages that matter for silent local printing.

The goal is not to make ClinicPlus manually know every printer. OpenPrintHub should identify the local OS printer, classify it, and return a recommended printing strategy to web applications.

## Capability Model

OpenPrintHub should expose these fields on each printer:

```json
{
  "id": "Xprinter_XP-420B",
  "name": "Xprinter XP-420B",
  "device_class": "label",
  "media_kind": "thermal-label",
  "command_sets": ["tspl", "zpl", "epl", "dpl"],
  "supported_job_types": ["raw", "image"],
  "preferred_label_language": "tspl",
  "preferred_job_type": "raw",
  "dpi": 203,
  "capability_source": "builtin-model-rule",
  "capability_confidence": "high"
}
```

Canonical values:

| Field | Values |
| --- | --- |
| `device_class` | `label`, `receipt`, `document`, `color-label`, `unknown` |
| `media_kind` | `thermal-label`, `receipt-roll`, `a4-document`, `color-roll-label`, `unknown` |
| `command_sets` | `tspl`, `zpl`, `epl`, `dpl`, `escpos`, `image`, `brother-raster`, `brother-ptouch`, `epson-esclabel`, `godex-ezpl`, `sato-sbpl`, `toshiba-tpcl`, `cab-jscript`, `honeywell-dp`, `unknown` |
| `preferred_label_language` | `tspl`, `zpl`, `image`, `none`, `unknown` |
| `preferred_job_type` | `raw`, `image`, `pdf`, `none` |
| `capability_source` | `override`, `builtin-exact-model`, `builtin-model-rule`, `vendor-family-rule`, `fallback` |
| `capability_confidence` | `high`, `medium`, `low` |

Selection rule for medication labels:

1. Use `tspl` when native TSPL / TSPL-EZD is present.
2. Use `zpl` when native ZPL/ZPL II or credible ZPL emulation is present.
3. Use `image` for DYMO LabelWriter, Brother QL, and other driver/raster-only label printers.
4. Do not show `document`, `receipt`, or `color-label` devices in the default medication-label printer picker.
5. Unknown printers are visible only behind an advanced option and default to `image`, not raw.

## Singapore-Market Model Catalog

### Xprinter

Singapore relevance: this is the currently installed local test printer (`Xprinter_XP-420B`). Xprinter models are also commonly sold through marketplace / distributor channels in Southeast Asia.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `Xprinter XP-420B`, `XP-421B`, `XP-410B`, `XP-450B`, `XP-470B` | `label` | `tspl` raw | Xprinter specs list TSPL plus ZPL/EPL/DPL emulation for XP-420B and nearby 4-inch models. Prefer TSPL because it is native for this family. |
| `XP-P441B` mobile label printer | `label` | `tspl` raw | Specs list TSPL/EPL/ZPL/DPL/CPCL/ESC-POS emulation. Treat as label, not receipt, when model name includes `P441B`. |
| `XP-N160`, `XP-Q200`, `XP-58`, generic Xprinter receipt models | `receipt` | `none` for medication labels | ESC/POS receipt printers must not appear in medication-label setup. |

Implementation patterns:

```text
(?i)xprinter.*xp[-_ ]?(420b|421b|410b|450b|470b|p441b)
(?i)xprinter.*(xp[-_ ]?n|xp[-_ ]?q|receipt|58|80)
```

Sources:

- Xprinter XP-420B official product page: https://www.xprintertech.com/xp-420b-thermal-label-printer.html
- Xprinter XP-421B brochure PDF: https://img5541.weyesimg.com/uploads/xprintertech.com/addon/17503293834108.pdf
- Xprinter XP-470B PDF: https://www.eet-tiskarny.cz/index.php?customfield_id=191&name=downloads_for_sale&option=com_virtuemart&view=plugin
- Xprinter XP-P441B PDF: https://d3m9l0v76dty0.cloudfront.net/system/photos/12733708/original/6fb53342c6f9eafdde3a90e6ce63a99e.pdf

### TSC

Singapore relevance: listed by Singapore suppliers such as LabelMark, Logicode, Tong Yong Labels, Anfotec, Shopee SG, and Lazada SG.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `TE200`, `TE210`, `TE300`, `TE310` | `label` | `tspl` raw | TSC TE series uses TSPL-EZD and is compatible with EPL/ZPL/ZPL II/DPL. |
| `TTP-244`, `TTP-247`, `TTP-225`, `TDP-225` | `label` | `tspl` raw | Common desktop TSC models. Prefer TSPL. |
| `DA200`, `DA210`, `DA220`, `DA300`, `DA310`, `DA320` | `label` | `tspl` raw | Direct thermal desktop family. Prefer TSPL. |
| `TX200`, `TX300`, `TC200`, `TC210`, `ME240`, `MX240`, `MX340P`, `MH240` | `label` | `tspl` raw | Higher-volume desktop/industrial models. Prefer TSPL. |

Implementation pattern:

```text
(?i)\btsc\b|ttp[-_ ]?(225|244|247)|tdp[-_ ]?225|te[23][01]0|tx[23]00|tc2|da[23]|me240|mx340|mh240
```

Sources:

- LabelMark Singapore TSC TE200/TE210: https://www.labelmark.com.sg/product/tsc-te200-series-2/
- Logicode Singapore TSC TE200 series: https://www.logicode.com.sg/products_uri/tsc-te200-series/
- Anfotec Singapore TSC TE210: https://anfotec.sg/products/tsc-te210-desktop-label-printer
- Tong Yong Labels Singapore TSC printer list: https://www.tongyonglabels.com.sg/tsc-printer/
- TSC TE series datasheet showing TSPL-EZD and ZPL/EPL/DPL compatibility: https://ipsiscan.com/wp-content/uploads/2025/10/IPSi-TSC-TE-Series-Datasheet.pdf
- TSC MH240 datasheet showing TSPL-EZD and ZPL/EPL/DPL compatibility: https://fs.tscprinters.com/system/files/mh240_series_eng_datasheet-a4.pdf

### Zebra

Singapore relevance: listed by Singapore suppliers such as Kingly, SourceIT, LabelMark, ICPM, Tong Yong Labels, Amazon SG, Lazada SG.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `ZD220`, `ZD230`, `ZD420`, `ZD421`, `ZD500`, `ZD621` | `label` | `zpl` raw | Modern Zebra desktop label printers. |
| `GK420d`, `GK420t`, `GX420`, `GX430`, `GC420`, `GT800` | `label` | `zpl` raw | Common older Zebra desktop families. |
| `ZT111`, `ZT231`, `ZT230`, `ZT411`, `ZT421`, `ZT510`, `ZT610`, `220Xi4` | `label` | `zpl` raw | Industrial models visible in Singapore supplier lists. |

Implementation pattern:

```text
(?i)zebra|zdesigner|zd[246]\d+|zt\d+|gk420|gx4[23]0|gc420|gt800|220xi
```

Sources:

- Kingly Singapore Zebra ZD421: https://www.kingly.sg/products/zebra-zd421-thermal-transfer-desktop-label-printer
- SourceIT Singapore Zebra ZD421: https://sourceit.com.sg/products/zebra-zd421-4-inch-desktop-printers-zd4a042-30pe00ez
- LabelMark Singapore Zebra ZD220: https://www.labelmark.com.sg/product/zebra-zd220/
- Tong Yong Labels Singapore Zebra printer list: https://www.tongyonglabels.com.sg/zebra-printer/
- Zebra ZD400 series spec says EPL and ZPL printer languages: https://www.zebra.com/us/en/products/spec-sheets/printers/desktop/zd400-series.html
- Zebra ZD421 technical specification PDF: https://www.zebra.cn/content/dam/zebra_dam/en/tech-specs/zd421-tech-specs-en-us.pdf

### Brother

Singapore relevance: Brother QL and TD models are sold through Singapore channels including Clover Global, Lionware, Pearlblue Tech, Kingly, Lazada SG, Amazon SG, and Brother Singapore support pages.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `Brother TD-4410D`, `TD-4420DN`, `TD-4520DN`, `TD-4550DNWB`, `TD-4420TN`, `TD-4520TN` | `label` | `zpl` raw when ZPL emulation is enabled | Brother TD-4D/TD-4T docs list raster, ESC/P, P-touch Template, ZPL emulation, CPCL emulation. Use ZPL only when model family supports it. |
| `Brother QL-800`, `QL-810W`, `QL-820NWB`, `QL-1110NWB` | `label` | `image` | QL series is best treated as raster/driver-backed. Do not assume ZPL. |
| `Brother MFC`, `DCP`, `HL`, office laser/inkjet | `document` | `none` for medication labels | A4/document printers should not appear in medication-label setup. |

Implementation patterns:

```text
(?i)brother.*td[-_ ]?(44|45|4[0-9]{3}|4t)|brother.*\b(tj|rj)\b
(?i)brother.*ql[-_ ]?(8|11)
(?i)brother.*(mfc|dcp|hl)
```

Sources:

- Clover Global Singapore Brother QL-800 / QL-820NWB: https://cloverglobal.com.sg/products/-original-brother-ql-800-ql-800-ql-820nwb-ql-820-professional-label-printer-wireless-labeler
- Pearlblue Tech Singapore Brother QL-820NWB: https://pearlbluetech.com.sg/products/brother-ql-820nwb-label-printer
- Brother Singapore TD-4410D support: https://www.brother.com.sg/en/support/td-4410d
- Kingly Singapore Brother TD-4420DN: https://www.kingly.sg/products/brother-td-4420dn-4-inch-network-industrial-label-printer
- Brother TD-4D series datasheet listing Raster, ESC/P, P-touch Template, ZPL emulation, CPCL emulation: https://www.brother.eu/-/media/Product-Downloads/Devices/Label-Printers/TD/TD-Series-Datasheet/TD-4D-series-datasheet-final.ashx
- Brother ZPL emulation guide: https://support.brother.com/g/b/files/dlf/docp100478/cv_td4t_eng_zpl_400.pdf
- Brother command reference model list: https://support.brother.com/g/s/es/dev/en/command/reference/index.html?navi=offall

### DYMO

Singapore relevance: DYMO LabelWriter 450 Turbo is sold by Singapore suppliers such as Kingly and is common in small clinics/offices.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `DYMO LabelWriter 400`, `450`, `450 Turbo`, `4XL`, `Wireless`, `550`, `550 Turbo`, `5XL` | `label` | `image` | DYMO is driver/SDK/raster-backed for our purposes. It should not receive TSPL/ZPL. |

Implementation pattern:

```text
(?i)dymo|labelwriter
```

Sources:

- Kingly Singapore DYMO LabelWriter 450 Turbo: https://www.kingly.sg/products/dymo-450-turbo
- DYMO SDK support page: https://www.dymo.com/online-support-sdk.html
- DYMO software/driver support page: https://www.dymo.com/support?cfid=user-guide

### BIXOLON

Singapore relevance: BIXOLON XD3-40d appears in Singapore POS workflows and EPOS Singapore supplies it as the barcode/label printer.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `XD3-40d`, `XD3-40t`, `XD3-40d-H`, `XD5-40d`, `XD5-40t`, `SLP-TX400`, `SLP-DX420`, `SRP-E770` | `label` | `zpl` raw when BPL-Z is supported | BIXOLON names ZPL compatibility as BPL-Z and EPL as BPL-E. Native SLCS is not in the ClinicPlus MVP renderer. |
| `SRP-330`, `SRP-350`, `SRP-380`, kitchen/receipt models | `receipt` | `none` for medication labels | ESC/POS receipt printers must not appear in medication-label setup. |

Implementation patterns:

```text
(?i)bixolon.*(xd3|xd5|slp[-_ ]?(tx|dx)|srp[-_ ]?e770)
(?i)bixolon.*(srp[-_ ]?3|receipt|kitchen)
```

Sources:

- EPOS Singapore label printing setup uses BIXOLON XD3-40d: https://www.epos.com.sg/knowledge-base-front-end/label-printing/
- EPOS Singapore hardware settings list BIXOLON XD3-40d for barcode labels: https://www.epos.com.sg/knowledge-base-front-end-hardware-settings/
- EPOS Singapore barcode printing guide says supplied barcode printer is typically BIXOLON-XD3-40d: https://www.epos.com.sg/knowledge-base-front-end-barcode-printing/
- BIXOLON XD3-40 product page: https://www.bixolon.com/product_view.php?idx=193
- BIXOLON XD3-40d support page: https://www.bixolon.com/download_view.php?idx=40
- BIXOLON EU XD3-40 page: https://bixoloneu.com/product/xd3-40-series/
- BIXOLON XD3-40d launch page lists SLCS, BPL-Z, BPL-E: https://bixoloneu.com/bixolon-launches-xd3-40d-desktop-label-printer-to-the-european-market/

### GoDEX

Singapore relevance: GoDEX GE300/G500 are sold by Singapore suppliers such as Anfotec, Kingly, and ElectronicsCrazy.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `GE300`, `GE330`, `G500`, `G530`, `DT4x`, `RT200`, `RT230` | `label` | `zpl` raw through GZPL | GoDEX supports EZPL/GEPL/GZPL/GDPL auto switching on many models. GZPL is the Zebra-compatible route for ClinicPlus MVP. |

Implementation pattern:

```text
(?i)godex|ge3[03]0|g5[03]0|dt4x|rt2[03]0
```

Sources:

- Anfotec Singapore GoDEX GE300: https://anfotec.sg/products/godex-ge300
- Kingly Singapore GoDEX GE300: https://www.kingly.sg/products/godex-ge300-thermal-transfer-desktop-label-printer
- ElectronicsCrazy Singapore GoDEX GE300 lists EZPL/GEPL/GZPL/GDPL auto switch: https://www.electronicscrazy.sg/godex-ge300-thermal-transfer-desktop-printer-with-ethernet-serial-and-usb-ports-included/
- GoDEX GE300/GE330 manual lists EZPL/GEPL/GZPL auto switch: https://www.godex.co.uk/downloads/Operating%20Manuals/UM_GE300-EN%20%281%29.pdf
- GoDEX GE300 brochure lists EZPL/GEPL/GZPL/GDPL auto switch: https://www.southwestscales.com/specsheets/Godex_GE300_Brochure_EN.pdf
- GoDEX emulation language note: https://www.godexintl.com/blog/15365491878411447

### Honeywell

Singapore relevance: Honeywell PC42T/PC42E-T/PM/PD/PX families are listed by Singapore suppliers such as LabelMark, Tong Yong Labels, Gamita Pak-IT, Lazada SG, and Carousell SG.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `PC42t`, `PC42t Plus`, `PC42E-T`, `PC42d` | `label` | `zpl` raw through ZSim/ZSim2 | Honeywell docs list ZSim/ZPL II plus EPL/DPL/Direct Protocol. |
| `PM42`, `PM45`, `PD45`, `PD45S`, `PX940` | `label` | `zpl` raw through ZSim/ZSim2 | Industrial models listed by Singapore suppliers. |

Implementation pattern:

```text
(?i)honeywell.*(pc42|pm42|pm45|pd45|px940)
```

Sources:

- LabelMark Singapore desktop printers include Honeywell PC42E-T and PC42T Plus: https://www.labelmark.com.sg/product-category/barcode-total-solutions/barcode-printer/entry-level-printers-desktop-printers/
- Tong Yong Labels Singapore Honeywell printer list: https://www.tongyonglabels.com.sg/honeywell-printer/
- Gamita Pak-IT Singapore Honeywell desktop printers: https://www.gamitapakit.com.sg/honeywell-desktop-printers.html
- Honeywell PC42t Plus page lists ESim, ZSim, DPL, Direct Protocol: https://automation.honeywell.com/us/en/products/productivity-solutions/printers/desktop-printers/pc42t-plus-desktop-thermal-transfer-barcode-printer
- Honeywell PC42E-T datasheet lists Direct Protocol, DPL, ZSim2, ESim: https://prod-edam.honeywell.com/content/dam/honeywell-edam/sps/ppr/en-us/public/products/printers/desktop/pc42e-t/documents/sps-pss-pc42e-t-printer-dts.pdf
- Honeywell PM45 page lists ZSim2/ZPL-II and other languages: https://automation.honeywell.com/us/en/products/productivity-solutions/printers/industrial-printers/pm45-industrial-printer
- Honeywell PD45/PD45S page lists FP/DP/IPL/ZPL/DPL: https://automation.honeywell.com/us/en/products/productivity-solutions/printers/industrial-printers/pd45s-pd45-industrial-printer

### Toshiba

Singapore relevance: Toshiba models are listed by Tong Yong Labels Singapore.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `B-EX4T1`, `B-EX4T2`, `B-EX4D2`, `B-EX6T`, `B-SX8T`, `B-852` | `label` | `zpl` raw when Z-Mode / ZPL emulation is enabled | Toshiba native TPCL is not in ClinicPlus MVP. ZPL-compatible mode is acceptable when available. |

Implementation pattern:

```text
(?i)toshiba.*(b[-_ ]?ex|b[-_ ]?sx|b[-_ ]?852)
```

Sources:

- Tong Yong Labels Singapore Toshiba printer list: https://www.tongyonglabels.com.sg/tsc-printer/
- Toshiba B-EX4T1 datasheet lists TPCL, ZPL II, DPL, BCI: https://www.toshibabusinessmea.com/wp-content/uploads/2021/02/BR_B-EX4T1_20210318-1.pdf
- Toshiba B-EX4T1 official page lists Z Mode ready and TPCL/BCI: https://www.toshibatec.eu/products/barcode-systems/b-ex4t1/
- Toshiba Z-Mode spec for ZPL II conversion: https://www.toshibatecstore.co.uk/documents/B-EX4%20Z-Mode%20Specification%201st%20Edition.pdf

### SATO

Singapore relevance: SATO is present in Asia Pacific and appears in Singapore marketplace/consumable compatibility lists. Treat as medium confidence until a site visit confirms exact model.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `WS2`, `WS4`, `CL4NX`, `CL6NX`, `CT4-LX`, `CT4-LX-HC`, `SA4` | `label` | `zpl` raw through SZPL when enabled | Native SBPL is not in ClinicPlus MVP. SZPL emulation is the path. |

Implementation pattern:

```text
(?i)sato.*(ws2|ws4|cl4nx|cl6nx|ct4|sa4)
```

Sources:

- SATO Asia Pacific WS4 datasheet lists SBPL and SZPL/SEPL/SDPL/SIPL emulations: https://satoasiapacific.com/wp-content/uploads/2019/03/WS4-Datasheet-FinalMar2024.pdf
- SATO supported emulation languages: https://sato-globalhelp.zendesk.com/hc/en-001/articles/38842183300377-Emulations-Languages-Supported-For-SATO-Printers

### cab

Singapore relevance: CAB models are listed by Tong Yong Labels Singapore.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `EOS2`, `EOS5`, `SQUIX`, `XC Q`, `XD Q` | `label` | `zpl` raw when ZPL emulation is enabled | Native cab JScript is not in ClinicPlus MVP. ZPL emulation is the path. |

Implementation pattern:

```text
(?i)\bcab\b|eos2|eos5|squix|xc q|xd q
```

Sources:

- Tong Yong Labels Singapore CAB printer list: https://www.tongyonglabels.com.sg/tsc-printer/
- cab support downloads list JScript and ZPL emulation documents: https://www.cab.de/en/support/support-downloads/?gruppierung=4&kategorie=32
- cab EOS2/EOS5 support downloads include ZPL emulation: https://www.cab.de/en/support/support-downloads/?bereich=45&produkt=17864&produktgruppe=48&ref=nav_lang&suchtyp=bereich

### Epson ColorWorks

Singapore relevance: Epson ColorWorks models are listed by Tong Yong Labels Singapore. They are color roll-label printers, not the likely first choice for medication stickers.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `CW-C4050`, `CW-C4000`, `CW-C6050`, `CW-C6500`, `CW-C8050` | `color-label` | `none` for medication labels | Epson uses ESC/Label. Do not send TSPL/ZPL. Exclude from medication-label setup until OpenPrintHub supports ESC/Label or app-level image/PDF for color label workflows. |

Implementation pattern:

```text
(?i)epson.*(colorworks|cw[-_ ]?c|c40|c60|c65|c80)
```

Sources:

- Tong Yong Labels Singapore Epson ColorWorks list: https://www.tongyonglabels.com.sg/tsc-printer/
- Epson ESC/Label command reference: https://files.support.epson.com/pdf/pos/bulk/esclabel_crg_en_07.pdf
- Epson CW-C4050 manual page referencing ESC/Label: https://support.epson.net/setupnavi/?LG2=C2&MKN=CW-C4050&OSC=MI&PINF=bsmanual

### A4 / Document Printers

These should never be shown as medication-label candidates by default.

Observed local examples from the test Mac:

| Models / patterns | Class | Recommended |
| --- | --- | --- |
| `Brother MFC L3770CDW series` | `document` | `none` |
| `HP LaserJet MFP M132snw` | `document` | `none` |
| `HP LaserJet MFP M141w` | `document` | `none` |
| `HP LaserJet Pro MFP M126nw` | `document` | `none` |

Implementation pattern:

```text
(?i)\b(hp|hewlett).*(laserjet|officejet|deskjet|mfp)
(?i)brother.*(mfc|dcp|hl)
(?i)(canon|epson).*(pixma|ecotank|workforce|office)
```

### Receipt / Kitchen Printers

These are common in Singapore POS environments but are not medication-label printers.

| Models / patterns | Class | Recommended | Notes |
| --- | --- | --- | --- |
| `Epson TM-T82`, `TM-T88`, `TM-m30` | `receipt` | `none` for medication labels | ESC/POS receipt roll. |
| `BIXOLON SRP-330`, `SRP-350` | `receipt` | `none` for medication labels | EPOS Singapore sells BIXOLON receipt/kitchen printers. |
| `Star TSP100`, `TSP650`, `mC-Print` | `receipt` | `none` for medication labels | ESC/POS/StarPRNT style. |
| Xprinter 58mm/80mm receipt models | `receipt` | `none` for medication labels | ESC/POS. |

Implementation pattern:

```text
(?i)(tm[-_ ]?t|tm[-_ ]?m|receipt|kitchen|srp[-_ ]?3|tsp100|tsp650|mc[-_ ]?print)
```

Sources:

- EPOS Singapore peripherals list includes BIXOLON receipt/kitchen/barcode printer categories: https://www.epos.com.sg/shop/peripherals/
- EPOS Singapore receipt printer page: https://www.epos.com.sg/product/bixolon-receipt-printer-srp330iiisk-epo/
- EPOS Singapore kitchen printer page: https://www.epos.com.sg/product/bixolon-kitchen-printer-srp330iiiesk-epo/

## MVP Implementation Rules

OpenPrintHub should evaluate capabilities in this order:

1. Site override file, for example `~/.openprinthub/printers.json`.
2. Exact normalized model names.
3. Built-in regex rules by vendor/model family.
4. Vendor fallback, for example `Zebra` means label/ZPL, `HP LaserJet` means document.
5. Unknown fallback.

When two rules match, more specific wins:

```text
exact-model > model-family > vendor-family > device-class-negative-rule > fallback
```

Negative rules should run before generic positive label rules for A4 and receipt printers. Example: `Brother MFC` must be `document`, even though `Brother TD` can be `label`.

First MVP exact rules:

```text
Xprinter XP-420B -> label, thermal-label, ["tspl","zpl","epl","dpl"], preferred tspl/raw, dpi 203
Brother MFC L3770CDW series -> document, a4-document, [], preferred none
HP LaserJet MFP M132snw -> document, a4-document, [], preferred none
HP LaserJet MFP M141w -> document, a4-document, [], preferred none
HP LaserJet Pro MFP M126nw -> document, a4-document, [], preferred none
```

First MVP family rules:

```text
Xprinter XP label family -> tspl/raw
TSC family -> tspl/raw
Zebra family -> zpl/raw
Brother TD/RJ/TJ family -> zpl/raw when supported
Brother QL family -> image
DYMO LabelWriter family -> image
BIXOLON XD/SLP label family -> zpl/raw when BPL-Z supported
GoDEX label family -> zpl/raw through GZPL
Honeywell PC/PM/PD/PX label family -> zpl/raw through ZSim
Toshiba B-EX/B-SX label family -> zpl/raw through Z-Mode
SATO label family -> zpl/raw through SZPL
cab label family -> zpl/raw through ZPL emulation
Epson ColorWorks -> color-label, excluded from medication label MVP
A4/document family -> excluded from medication label setup
Receipt/kitchen family -> excluded from medication label setup
```

## ClinicPlus Consumption Contract

ClinicPlus should not keep its own copy of this catalog. It should call OpenPrintHub and use the returned fields:

- Default printer picker filter: `device_class === "label"` and `preferred_label_language !== "none"`.
- Default language: `preferred_label_language`.
- Test print:
  - `tspl` / `zpl` -> render command set and send `type: "raw"`.
  - `image` -> render PNG and send `type: "image"`.
- A4/document printers are visible only in an advanced "Show non-label printers" list.
- Unknown printers are visible only in advanced mode and default to image, never raw.

## Maintenance Process

When a new clinic reports a printer:

1. Capture OS printer name from OpenPrintHub `/v1/printers`.
2. Capture vendor/model from `lpstat -v`, Windows printer properties, or USB device info if available.
3. Search for official datasheet/manual for command languages.
4. Search Singapore supplier/marketplace only to prove local availability.
5. Add exact model rule if the model is common or already seen at a clinic.
6. Add family rule only when the command language is consistent across the family.
7. If uncertain, classify as `label` with `preferred_label_language: "image"` or `unknown`, not raw.

## Open Questions

- Whether OpenPrintHub should expose a separate `/v1/printers/:id/capabilities` endpoint in addition to embedding capabilities in `/v1/printers`.
- Whether to support native non-ZPL languages later: `brother-raster`, `epson-esclabel`, `sato-sbpl`, `toshiba-tpcl`, `cab-jscript`, `godex-ezpl`, `honeywell-dp`.
- Whether to add active probing for network printers. USB/CUPS names often lack enough detail, so built-in rules plus overrides may be safer for MVP.
