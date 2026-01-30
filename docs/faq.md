# Frequently Asked Questions (FAQ)

## Basic Questions

### Q: What is OpenPrintHub?

OpenPrintHub is a locally-running print service that allows web applications to achieve silent printing (without user confirmation) through API calls. It is an open-source alternative to commercial solutions like JSPrintManager.

### Q: Which operating systems are supported?

- ✅ macOS 10.15+ (Intel and Apple Silicon)
- ✅ Windows 10+
- 🚧 Linux (planned)

### Q: Is it free?

Yes, OpenPrintHub is open source under the MIT license and can be used freely for personal and commercial projects.

---

## Installation Issues

### Q: macOS says "Cannot be opened because the developer cannot be verified"

This is macOS's security mechanism. Solutions:

1. Right-click the app → Open
2. Or allow it in System Preferences → Security & Privacy

Command-line method:

```bash
xattr -d com.apple.quarantine ./oph-darwin-*
```

### Q: Windows Firewall blocked the program

On first run, Windows may show a firewall prompt. Click "Allow access".

Or manually add a rule:

```powershell
netsh advfirewall firewall add rule name="OpenPrintHub" dir=in action=allow protocol=TCP localport=16800
```

---

## Printing Issues

### Q: Printer list is empty

1. **Check printer connection** - Ensure the printer is powered on and connected
2. **Check driver installation** - Ensure system has printer drivers installed
3. **Restart the service** - Restart OpenPrintHub

Check system printers:

```bash
# macOS
lpstat -p

# Windows PowerShell
Get-Printer
```

### Q: Print job status stays "queued"

Possible causes:
- Printer is offline
- Print queue is blocked
- Printer driver issues

Solutions:
1. Check printer status
2. Clear system print queue
3. Restart the printer

### Q: PDF prints blank

1. Verify the PDF file is valid
2. Check if Base64 encoding is correct
3. Try opening the PDF with the system's default viewer to confirm content is correct

### Q: ESC/POS commands not working

1. Confirm the printer supports ESC/POS
2. Use `type: "raw"` when sending
3. Verify command format is correct

---

## API Issues

### Q: CORS error

Ensure OpenPrintHub is started with correct CORS origins configured:

```bash
# Development
./oph -cors "*"

# Production
./oph -cors "https://your-app.com"
```

### Q: Connection Refused

1. Verify OpenPrintHub is running
2. Confirm the port is correct (default 16800)
3. Check firewall settings

### Q: WebSocket disconnects frequently

WebSocket connections may disconnect due to network issues. Implement auto-reconnection:

```javascript
function connect() {
  const ws = new WebSocket('ws://localhost:16800/v1/ws');
  ws.onclose = () => setTimeout(connect, 3000);
  return ws;
}
```

---

## Integration Questions

### Q: How to use in a React project?

```jsx
import { useState, useEffect } from 'react';

function usePrinters() {
  const [printers, setPrinters] = useState([]);
  
  useEffect(() => {
    fetch('http://localhost:16800/v1/printers')
      .then(res => res.json())
      .then(setPrinters)
      .catch(console.error);
  }, []);
  
  return printers;
}

function PrintButton({ pdfData }) {
  const printers = usePrinters();
  
  const print = async () => {
    const response = await fetch('http://localhost:16800/v1/print', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        printer: printers[0]?.name,
        type: 'pdf',
        data: pdfData,
        settings: { copies: 1 }
      })
    });
    const result = await response.json();
    console.log('Job ID:', result.job_id);
  };
  
  return <button onClick={print}>Print</button>;
}
```

### Q: How to use in a Vue project?

```vue
<script setup>
import { ref, onMounted } from 'vue';

const printers = ref([]);

onMounted(async () => {
  const res = await fetch('http://localhost:16800/v1/printers');
  printers.value = await res.json();
});

async function print(pdfBase64) {
  const res = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printers.value[0]?.name,
      type: 'pdf',
      data: pdfBase64
    })
  });
  return res.json();
}
</script>
```

### Q: How to detect if OpenPrintHub is running?

```javascript
async function isOPHRunning() {
  try {
    const res = await fetch('http://localhost:16800/health', {
      signal: AbortSignal.timeout(2000)
    });
    return res.ok;
  } catch {
    return false;
  }
}

// Usage
if (await isOPHRunning()) {
  console.log('OpenPrintHub is ready');
} else {
  console.log('Please start OpenPrintHub');
}
```

---

## Security Questions

### Q: Is it secure?

OpenPrintHub only listens to local requests and is not exposed to the public internet. However, it's still recommended to:
- Configure explicit CORS origins in production
- Do not expose OpenPrintHub to the public internet

### Q: How to restrict access sources?

Use the `-cors` argument to limit allowed domains:

```bash
./oph -cors "https://trusted-app.com"
```

---

## Other Questions

### Q: How to update to a new version?

1. Stop the currently running OpenPrintHub
2. Download the new version binary
3. Replace the old file
4. Restart

### Q: How to check the current version?

```bash
./oph -version
```

### Q: Where to report bugs?

Please submit issues on GitHub:
https://github.com/medpath1024/OpenPrintHub/issues

When submitting, please include:
- Operating system version
- OpenPrintHub version
- Error logs
- Steps to reproduce
