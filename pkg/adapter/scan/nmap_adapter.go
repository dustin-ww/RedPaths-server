package scan

import (
	"RedPaths-server/pkg/adapter/serializable"
	"RedPaths-server/pkg/adapter/util"
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/model"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
)

type NmapAdapter struct {
	*util.ExecutableHelper
	version string
}

type NmapScanOptions struct {
	interfaces.ScanOptions
	ServiceScan    bool
	ScriptScan     bool
	UDPScan        bool
	PortRange      string
	TimingTemplate int  // -T0 bis -T5
	AggressiveScan bool // -A
	OSScan         bool // -O
	HostDiscovery  bool // -sn
	TracerouteScan bool // --traceroute
	DNSResolution  bool // -n
	IPv6Scan       bool // -6
}

type NmapScanResult struct {
	Raw    []byte
	Data   serializable.NmapResult
	xmlDoc *xmlquery.Node
}

func (r *NmapScanResult) GetXMLDocument() (*xmlquery.Node, error) {
	if r.xmlDoc == nil {
		doc, err := xmlquery.Parse(strings.NewReader(string(r.Raw)))
		if err != nil {
			return nil, fmt.Errorf("fehler beim Parsen des XML-Dokuments: %w", err)
		}
		r.xmlDoc = doc
	}
	return r.xmlDoc, nil
}

func (r *NmapScanResult) QueryValue(xpath string) (string, error) {
	doc, err := r.GetXMLDocument()
	if err != nil {
		return "", err
	}

	node := xmlquery.FindOne(doc, xpath)
	if node == nil {
		return "", nil
	}
	return node.InnerText(), nil
}

func (r *NmapScanResult) QueryNodes(xpath string) ([]*xmlquery.Node, error) {
	doc, err := r.GetXMLDocument()
	if err != nil {
		return nil, err
	}

	return xmlquery.Find(doc, xpath), nil
}

func (r *NmapScanResult) GetRawOutput() []byte {
	return r.Raw
}

// NmapScanResult-spezifische Methoden
type NmapScanResultExtended interface {
	interfaces.ScanResult
	GetNmapResult() serializable.NmapResult
}

func (r *NmapScanResult) GetNmapResult() serializable.NmapResult {
	return r.Data
}

func (r *NmapScanResult) GetHosts() []model.Host {
	var hosts []model.Host

	for _, nmapHost := range r.Data.Host {
		hostBuilder := model.NewHostBuilder()

		for _, addr := range nmapHost.Address {
			if addr.Addrtype == "ipv4" {
				hostBuilder.WithIP(addr.Addr)
				break
			}
		}

		hostBuilder.WithName("Unknown")

		host, err := hostBuilder.Build()
		if err != nil {
			log.Printf("Fehler beim Erstellen des Hosts: %v", err)
			continue
		}

		hosts = append(hosts, *host)
	}

	return hosts
}

func (r *NmapScanResult) GetServices() []model.Service {
	var services []model.Service

	for _, nmapHost := range r.Data.Host {

		for _, addr := range nmapHost.Address {
			if addr.Addrtype == "ipv4" {
				break
			}
		}

		for _, port := range nmapHost.Ports.Port {
			if port.State.State == "open" {
				serviceBuilder := model.NewServiceBuilder()
				serviceBuilder.WithName(port.Service.Name)
				serviceBuilder.WithPort(port.Portid)

				//if port.Protocol != "" {
				//	serviceBuilder.WithProtocol(port.Protocol)
				//}

				service := serviceBuilder.Build()
				services = append(services, service)
			}
		}
	}

	return services
}

func NewNmapAdapter() interfaces.ScanAdapter {
	return &NmapAdapter{
		ExecutableHelper: util.NewExecutableHelper("nmap"),
	}
}

func (n *NmapAdapter) GetName() string {
	return "nmap"
}

func (n *NmapAdapter) GetVersion() string {
	if n.version == "" {
		output, err := util.ExecWithFallback(context.Background(), n, "nmap", "--version")
		if err == nil {
			parts := strings.Split(string(output), " ")
			if len(parts) > 2 {
				n.version = strings.TrimSpace(parts[2])
			}
		}
	}
	return n.version
}

func (n *NmapAdapter) IsAvailable(ctx context.Context) bool {
	_, err := util.ExecWithFallback(ctx, n, "nmap", "--version")
	return err == nil
}

func (n *NmapAdapter) Scan(ctx context.Context, options ...interfaces.ScanOption) (interfaces.ScanResult, error) {
	opts := &NmapScanOptions{
		ScanOptions: interfaces.ScanOptions{
			Timeout:      30 * time.Minute,
			OutputFormat: "xml",
		},
		DNSResolution: true,
	}

	for _, option := range options {
		option(opts)
	}

	if len(opts.Targets) == 0 {
		return nil, errors.New("no targets specified")
	}

	var cancel context.CancelFunc
	if ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	args := []string{}

	args = append(args, "-oX", "-")

	if opts.PortRange != "" {
		args = append(args, "-p", opts.PortRange)
	}

	if opts.ServiceScan {
		args = append(args, "-sV")
	}

	if opts.ScriptScan {
		args = append(args, "-sC")
	}

	if opts.UDPScan {
		args = append(args, "-sU")
	}

	if opts.TimingTemplate > 0 {
		args = append(args, fmt.Sprintf("-T%d", opts.TimingTemplate))
	}

	if opts.AggressiveScan {
		args = append(args, "-A")
	}

	if opts.OSScan {
		args = append(args, "-O")
	}

	if !opts.HostDiscovery {
		args = append(args, "-Pn")
	}

	if opts.TracerouteScan {
		args = append(args, "--traceroute")
	}

	if !opts.DNSResolution {
		args = append(args, "-n")
	}

	if opts.IPv6Scan {
		args = append(args, "-6")
	}

	args = append(args, opts.CustomFlags...)

	args = append(args, opts.Targets...)

	log.Printf("Executing nmap with args: %v", args)

	//output, err := util.ExecWithFallback(ctx, n, "nmap", args...)

	//TODO REMOVE - Only for local Testing
	xmlFilePath := "/workspace/assets/nmap_homelab.xml"
	output, err := os.ReadFile(xmlFilePath)

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("nmap scan timed out after %v", opts.Timeout)
		}
		return nil, fmt.Errorf("nmap execution failed: %w, output: %s", err, string(output))
	}

	var nmapResult serializable.NmapResult
	if err := xml.Unmarshal(output, &nmapResult); err != nil {
		return nil, fmt.Errorf("failed to parse nmap XML output: %w", err)
	}

	return &NmapScanResult{
		Raw:  output,
		Data: nmapResult,
	}, nil
}

func WithTargets(targets []string) interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			nmapOpts.Targets = targets
		}
	}
}

func WithServiceScan() interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			nmapOpts.ServiceScan = true
		}
	}
}

func WithScriptScan() interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			nmapOpts.ScriptScan = true
		}
	}
}

func WithUDPScan() interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			nmapOpts.UDPScan = true
		}
	}
}

func WithPortRange(ports string) interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			nmapOpts.PortRange = ports
		}
	}
}

func WithTimingTemplate(level int) interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			if level >= 0 && level <= 5 {
				nmapOpts.TimingTemplate = level
			}
		}
	}
}

func WithAggressiveScan() interfaces.ScanOption {
	return func(opts interface{}) {
		if nmapOpts, ok := opts.(*NmapScanOptions); ok {
			nmapOpts.AggressiveScan = true
		}
	}
}
