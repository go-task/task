package taskfile

import "time"

// WithInsecure allows the [Reader] to make insecure connections when reading
// remote taskfiles. By default, insecure connections are rejected.
func WithInsecure(insecure bool) *insecureOption {
	return &insecureOption{insecure: insecure}
}

type insecureOption struct {
	insecure bool
}

func (o *insecureOption) ApplyToReader(r *Reader) {
	r.insecure = o.insecure
}

// WithDownload forces the [Reader] to download a fresh copy of the taskfile
// from the remote source.
func WithDownload(download bool) *downloadOption {
	return &downloadOption{download: download}
}

type downloadOption struct {
	download bool
}

func (o *downloadOption) ApplyToReader(r *Reader) {
	r.download = o.download
}

// WithOffline stops the [Reader] from being able to make network connections.
// It will still be able to read local files and cached copies of remote files.
func WithOffline(offline bool) *offlineOption {
	return &offlineOption{offline: offline}
}

type offlineOption struct {
	offline bool
}

func (o *offlineOption) ApplyToReader(r *Reader) {
	r.offline = o.offline
}

// WithTrustedHosts configures the [Reader] with a list of trusted hosts for remote
// Taskfiles. Hosts in this list will not prompt for user confirmation.
func WithTrustedHosts(trustedHosts []string) *trustedHostsOption {
	return &trustedHostsOption{trustedHosts: trustedHosts}
}

type trustedHostsOption struct {
	trustedHosts []string
}

func (o *trustedHostsOption) ApplyToReader(r *Reader) {
	r.trustedHosts = o.trustedHosts
}

// WithTempDir sets the temporary directory that will be used by the [Reader].
// By default, the reader uses [os.TempDir].
func WithTempDir(tempDir string) *tempDirOption {
	return &tempDirOption{tempDir: tempDir}
}

type tempDirOption struct {
	tempDir string
}

func (o *tempDirOption) ApplyToReader(r *Reader) {
	r.tempDir = o.tempDir
}

// WithCacheExpiryDuration sets the duration after which the cache is considered
// expired. By default, the cache is considered expired after 24 hours.
func WithCacheExpiryDuration(duration time.Duration) *cacheExpiryDurationOption {
	return &cacheExpiryDurationOption{duration: duration}
}

type cacheExpiryDurationOption struct {
	duration time.Duration
}

func (o *cacheExpiryDurationOption) ApplyToReader(r *Reader) {
	r.cacheExpiryDuration = o.duration
}

// WithDebugFunc sets the debug function to be used by the [Reader]. If set,
// this function will be called with debug messages. This can be useful if the
// caller wants to log debug messages from the [Reader]. By default, no debug
// function is set and the logs are not written.
func WithDebugFunc(debugFunc DebugFunc) *debugFuncOption {
	return &debugFuncOption{debugFunc: debugFunc}
}

type debugFuncOption struct {
	debugFunc DebugFunc
}

func (o *debugFuncOption) ApplyToReader(r *Reader) {
	r.debugFunc = o.debugFunc
}

// WithPromptFunc sets the prompt function to be used by the [Reader]. If set,
// this function will be called with prompt messages. The function should
// optionally log the message to the user and return nil if the prompt is
// accepted and the execution should continue. Otherwise, it should return an
// error which describes why the prompt was rejected. This can then be caught
// and used later when calling the [Reader.Read] method. By default, no prompt
// function is set and all prompts are automatically accepted.
func WithPromptFunc(promptFunc PromptFunc) *promptFuncOption {
	return &promptFuncOption{promptFunc: promptFunc}
}

type promptFuncOption struct {
	promptFunc PromptFunc
}

func (o *promptFuncOption) ApplyToReader(r *Reader) {
	r.promptFunc = o.promptFunc
}

// WithCACert sets the path to a custom CA certificate for TLS connections.
func WithCACert(caCert string) *caCertOption {
	return &caCertOption{caCert: caCert}
}

type caCertOption struct {
	caCert string
}

func (o *caCertOption) ApplyToReader(r *Reader) {
	r.caCert = o.caCert
}

func (o *caCertOption) ApplyToBaseNode(n *baseNode) {
	n.caCert = o.caCert
}

// WithCert sets the path to a client certificate for TLS connections.
func WithCert(cert string) *certOption {
	return &certOption{cert: cert}
}

type certOption struct {
	cert string
}

func (o *certOption) ApplyToReader(r *Reader) {
	r.cert = o.cert
}

func (o *certOption) ApplyToBaseNode(n *baseNode) {
	n.cert = o.cert
}

// WithCertKey sets the path to a client certificate key for TLS connections.
func WithCertKey(certKey string) *certKeyOption {
	return &certKeyOption{certKey: certKey}
}

type certKeyOption struct {
	certKey string
}

func (o *certKeyOption) ApplyToReader(r *Reader) {
	r.certKey = o.certKey
}

func (o *certKeyOption) ApplyToBaseNode(n *baseNode) {
	n.certKey = o.certKey
}

// WithReaderHeaders sets the HTTP headers to send to each configured host.
func WithReaderHeaders(headersByHost HeadersByHost) *readerHeadersOption {
	return &readerHeadersOption{headersByHost: headersByHost}
}

type readerHeadersOption struct {
	headersByHost HeadersByHost
}

func (o *readerHeadersOption) ApplyToReader(r *Reader) {
	r.headersByHost = o.headersByHost
}

// WithParent sets the parent node for the node base.
func WithParent(parent Node) *parentOption {
	return &parentOption{parent: parent}
}

type parentOption struct {
	parent Node
}

func (o *parentOption) ApplyToBaseNode(n *baseNode) {
	n.parent = o.parent
}

// WithChecksum sets the checksum for the node base.
func WithChecksum(checksum string) *checksumOption {
	return &checksumOption{checksum: checksum}
}

type checksumOption struct {
	checksum string
}

func (o *checksumOption) ApplyToBaseNode(n *baseNode) {
	n.checksum = o.checksum
}

// WithHeaders sets the HTTP headers to send, keyed by host.
func WithHeaders(headersByHost HeadersByHost) *headersOption {
	return &headersOption{headersByHost: headersByHost}
}

type headersOption struct {
	headersByHost HeadersByHost
}

func (o *headersOption) ApplyToBaseNode(n *baseNode) {
	n.headersByHost = o.headersByHost
}

// WithLine specifies the line number that the [Snippet] should center around
// and point to.
func WithLine(line int) *lineOption {
	return &lineOption{line: line}
}

type lineOption struct {
	line int
}

func (o *lineOption) ApplyToSnippet(s *Snippet) {
	s.line = o.line
}

// WithColumn specifies the column number that the [Snippet] should point to.
func WithColumn(column int) *columnOption {
	return &columnOption{column: column}
}

type columnOption struct {
	column int
}

func (o *columnOption) ApplyToSnippet(s *Snippet) {
	s.column = o.column
}

// WithPadding specifies the number of lines to include before and after the
// selected line in the [Snippet].
func WithPadding(padding int) *paddingOption {
	return &paddingOption{padding: padding}
}

type paddingOption struct {
	padding int
}

func (o *paddingOption) ApplyToSnippet(s *Snippet) {
	s.padding = o.padding
}

// WithNoIndicators specifies that the [Snippet] should not include line or
// column indicators.
func WithNoIndicators() *noIndicatorsOption {
	return &noIndicatorsOption{}
}

type noIndicatorsOption struct{}

func (o *noIndicatorsOption) ApplyToSnippet(s *Snippet) {
	s.noIndicators = true
}
