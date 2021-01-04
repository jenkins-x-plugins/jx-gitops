---
title: API Documentation
linktitle: API Documentation
description: Reference of the jx-promote configuration
weight: 10
---
<p>Packages:</p>
<ul>
<li>
<a href="#gitops.jenkins-x.io%2fv1alpha1">gitops.jenkins-x.io/v1alpha1</a>
</li>
</ul>
<h2 id="gitops.jenkins-x.io/v1alpha1">gitops.jenkins-x.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#gitops.jenkins-x.io/v1alpha1.KptStrategies">KptStrategies</a>
</li><li>
<a href="#gitops.jenkins-x.io/v1alpha1.PipelineCatalog">PipelineCatalog</a>
</li><li>
<a href="#gitops.jenkins-x.io/v1alpha1.Quickstarts">Quickstarts</a>
</li><li>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMapping">SecretMapping</a>
</li><li>
<a href="#gitops.jenkins-x.io/v1alpha1.SourceConfig">SourceConfig</a>
</li></ul>
<h3 id="gitops.jenkins-x.io/v1alpha1.KptStrategies">KptStrategies
</h3>
<p>
<p>KptStrategies contains a collection of merge strategies Jenkins X will use when performing kpt updates</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
gitops.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>KptStrategies</code></td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.KptStrategyConfig">
[]KptStrategyConfig
</a>
</em>
</td>
<td>
<p>KptStrategyConfig contains a collection of merge strategies Jenkins X will use when performing kpt updates</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.PipelineCatalog">PipelineCatalog
</h3>
<p>
<p>PipelineCatalog represents a collection quickstart project</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
gitops.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>PipelineCatalog</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.PipelineCatalogSpec">
PipelineCatalogSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the PipelineCatalog from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>repositories</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.PipelineCatalogSource">
[]PipelineCatalogSource
</a>
</em>
</td>
<td>
<p>Repositories the repositories containing pipeline catalogs</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.Quickstarts">Quickstarts
</h3>
<p>
<p>Quickstarts represents a collection quickstart project</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
gitops.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Quickstarts</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartsSpec">
QuickstartsSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the specified quicksatrt configuration</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>quickstarts</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartSource">
[]QuickstartSource
</a>
</em>
</td>
<td>
<p>Quickstarts custom quickstarts to include</p>
</td>
</tr>
<tr>
<td>
<code>defaultOwner</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultOwner the default owner if not specfied</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartImport">
[]QuickstartImport
</a>
</em>
</td>
<td>
<p>Imports import quickstarts from the version stream</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.SecretMapping">SecretMapping
</h3>
<p>
<p>SecretMapping represents a collection of mappings of Secrets to destinations in the underlying secret store (e.g. Vault keys)</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
gitops.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>SecretMapping</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMappingSpec">
SecretMappingSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the SecretMapping from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>secrets</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretRule">
[]SecretRule
</a>
</em>
</td>
<td>
<p>Secrets rules for each secret</p>
</td>
</tr>
<tr>
<td>
<code>defaults</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.Defaults">
Defaults
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.SourceConfig">SourceConfig
</h3>
<p>
<p>SourceConfig represents a collection source repostory groups and repositories</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
gitops.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>SourceConfig</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.SourceConfigSpec">
SourceConfigSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the SourceConfig from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>groups</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.RepositoryGroup">
[]RepositoryGroup
</a>
</em>
</td>
<td>
<p>Groups the groups of source repositories</p>
</td>
</tr>
<tr>
<td>
<code>scheduler</code></br>
<em>
string
</em>
</td>
<td>
<p>Scheduler the default scheduler for any group/repository which does not specify one</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsServers</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.JenkinsServer">
[]JenkinsServer
</a>
</em>
</td>
<td>
<p>JenkinsServers the jenkins servers configured for this repository</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsFolderTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsFolderTemplate the default template file to use to generate the folder job DSL script</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsJobTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsJobTemplate the default template file to use to generate the projects job DSL script</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.BackendType">BackendType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.Defaults">Defaults</a>, 
<a href="#gitops.jenkins-x.io/v1alpha1.SecretRule">SecretRule</a>)
</p>
<p>
<p>BackendType describes a secrets backend</p>
</p>
<h3 id="gitops.jenkins-x.io/v1alpha1.Defaults">Defaults
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMappingSpec">SecretMappingSpec</a>)
</p>
<p>
<p>Defaults contains default mapping configuration for any Kubernetes secrets to External Secrets</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>backendType</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.BackendType">
BackendType
</a>
</em>
</td>
<td>
<p>DefaultBackendType the default back end to use if there&rsquo;s no specific mapping</p>
</td>
</tr>
<tr>
<td>
<code>gcpSecretsManager</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.GcpSecretsManager">
GcpSecretsManager
</a>
</em>
</td>
<td>
<p>GcpSecretsManager config</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.GcpSecretsManager">GcpSecretsManager
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.Defaults">Defaults</a>, 
<a href="#gitops.jenkins-x.io/v1alpha1.SecretRule">SecretRule</a>)
</p>
<p>
<p>GcpSecretsManager the predicates which must be true to invoke the associated tasks/pipelines</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version of the referenced secret</p>
</td>
</tr>
<tr>
<td>
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectId for the secret, defaults to the current GCP project</p>
</td>
</tr>
<tr>
<td>
<code>uniquePrefix</code></br>
<em>
string
</em>
</td>
<td>
<p>UniquePrefix needs to be a unique prefix in the GCP project where the secret resides, defaults to cluster name</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.JenkinsServer">JenkinsServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SourceConfigSpec">SourceConfigSpec</a>)
</p>
<p>
<p>JenkinsServer the Jenkins server configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>server</code></br>
<em>
string
</em>
</td>
<td>
<p>Server the name of the Jenkins Server to use</p>
</td>
</tr>
<tr>
<td>
<code>folderTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>FolderTemplate the default template file to use to generate the folder job DSL script</p>
</td>
</tr>
<tr>
<td>
<code>jobTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JobTemplate the default template file to use to generate the projects job DSL script</p>
</td>
</tr>
<tr>
<td>
<code>groups</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.RepositoryGroup">
[]RepositoryGroup
</a>
</em>
</td>
<td>
<p>Groups the groups of source repositories</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.KptStrategyConfig">KptStrategyConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.KptStrategies">KptStrategies</a>)
</p>
<p>
<p>KptStrategyConfig used by jx gitops upgrade kpt</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>relativePath</code></br>
<em>
string
</em>
</td>
<td>
<p>RelativePath the relative path to the folder the strategy should apply to</p>
</td>
</tr>
<tr>
<td>
<code>strategy</code></br>
<em>
string
</em>
</td>
<td>
<p>Strategy is the merge strategy kpt will use see <a href="https://googlecontainertools.github.io/kpt/reference/pkg/update/#flags">https://googlecontainertools.github.io/kpt/reference/pkg/update/#flags</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.Mapping">Mapping
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretRule">SecretRule</a>)
</p>
<p>
<p>Mapping the predicates which must be true to invoke the associated tasks/pipelines</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name the secret entry name which maps to the Key of the Secret.Data map</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Key the Vault key to load the secret value</p>
</td>
</tr>
<tr>
<td>
<code>property</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Property the Vault property on the key to load the secret value</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.PipelineCatalogSource">PipelineCatalogSource
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.PipelineCatalogSpec">PipelineCatalogSpec</a>)
</p>
<p>
<p>PipelineCatalogSource the source of a pipeline catalog</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>label</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gitUrl</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gitRef</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.PipelineCatalogSpec">PipelineCatalogSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.PipelineCatalog">PipelineCatalog</a>)
</p>
<p>
<p>PipelineCatalogSpec defines the desired state of PipelineCatalog.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>repositories</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.PipelineCatalogSource">
[]PipelineCatalogSource
</a>
</em>
</td>
<td>
<p>Repositories the repositories containing pipeline catalogs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.QuickstartImport">QuickstartImport
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartsSpec">QuickstartsSpec</a>)
</p>
<p>
<p>QuickstartImport imports quickstats from another folder (such as from the shared version stream)</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>file</code></br>
<em>
string
</em>
</td>
<td>
<p>File file name relative to the root directory to load</p>
</td>
</tr>
<tr>
<td>
<code>includes</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>excludes</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.QuickstartSource">QuickstartSource
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartsSpec">QuickstartsSpec</a>)
</p>
<p>
<p>QuickstartSource the source of a quickstart</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Owner</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Version</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Language</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Framework</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Tags</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>DownloadZipURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>GitServer</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>GitKind</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.QuickstartsSpec">QuickstartsSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.Quickstarts">Quickstarts</a>)
</p>
<p>
<p>QuickstartsSpec defines the desired state of Quickstarts.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>quickstarts</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartSource">
[]QuickstartSource
</a>
</em>
</td>
<td>
<p>Quickstarts custom quickstarts to include</p>
</td>
</tr>
<tr>
<td>
<code>defaultOwner</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultOwner the default owner if not specfied</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.QuickstartImport">
[]QuickstartImport
</a>
</em>
</td>
<td>
<p>Imports import quickstarts from the version stream</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.Repository">Repository
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.RepositoryGroup">RepositoryGroup</a>)
</p>
<p>
<p>Repository the name of the repository to import and the optional scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name the name of the repository</p>
</td>
</tr>
<tr>
<td>
<code>scheduler</code></br>
<em>
string
</em>
</td>
<td>
<p>Scheduler the optional name of the scheduler to use if different to the group</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsJobTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsJobTemplate the template file to use to generate the projects job DSL script</p>
</td>
</tr>
<tr>
<td>
<code>description</code></br>
<em>
string
</em>
</td>
<td>
<p>Description the optional description of this repository</p>
</td>
</tr>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL the URL to access this repository</p>
</td>
</tr>
<tr>
<td>
<code>httpCloneURL</code></br>
<em>
string
</em>
</td>
<td>
<p>HTTPCloneURL the HTTP/HTTPS based clone URL</p>
</td>
</tr>
<tr>
<td>
<code>sshCloneURL</code></br>
<em>
string
</em>
</td>
<td>
<p>SSHCloneURL the SSH based clone URL</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.RepositoryGroup">RepositoryGroup
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.JenkinsServer">JenkinsServer</a>, 
<a href="#gitops.jenkins-x.io/v1alpha1.SourceConfigSpec">SourceConfigSpec</a>)
</p>
<p>
<p>SourceConfigSpec defines the desired state of SourceConfig.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>provider</code></br>
<em>
string
</em>
</td>
<td>
<p>Provider the git provider server URL</p>
</td>
</tr>
<tr>
<td>
<code>providerKind</code></br>
<em>
string
</em>
</td>
<td>
<p>ProviderKind the git provider kind</p>
</td>
</tr>
<tr>
<td>
<code>providerName</code></br>
<em>
string
</em>
</td>
<td>
<p>ProviderName the git provider name</p>
</td>
</tr>
<tr>
<td>
<code>owner</code></br>
<em>
string
</em>
</td>
<td>
<p>Owner the name of the organisation/owner/project/user that owns the repository</p>
</td>
</tr>
<tr>
<td>
<code>repositories</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.Repository">
[]Repository
</a>
</em>
</td>
<td>
<p>Repositories the repositories for the</p>
</td>
</tr>
<tr>
<td>
<code>scheduler</code></br>
<em>
string
</em>
</td>
<td>
<p>Scheduler the default scheduler for this group</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsFolderTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsFolderTemplate the default template file to use to generate the folder job DSL script</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsJobTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsJobTemplate the default job template file to use to generate the projects job DSL script</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.SecretMappingSpec">SecretMappingSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMapping">SecretMapping</a>)
</p>
<p>
<p>SecretMappingSpec defines the desired state of SecretMapping.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secrets</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretRule">
[]SecretRule
</a>
</em>
</td>
<td>
<p>Secrets rules for each secret</p>
</td>
</tr>
<tr>
<td>
<code>defaults</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.Defaults">
Defaults
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.SecretRule">SecretRule
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMappingSpec">SecretMappingSpec</a>)
</p>
<p>
<p>SecretRule the rules for a specific Secret</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name name of the secret</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>Namespace name of the secret</p>
</td>
</tr>
<tr>
<td>
<code>backendType</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.BackendType">
BackendType
</a>
</em>
</td>
<td>
<p>BackendType for the secret</p>
</td>
</tr>
<tr>
<td>
<code>mappings</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.Mapping">
[]Mapping
</a>
</em>
</td>
<td>
<p>Mappings one more mappings</p>
</td>
</tr>
<tr>
<td>
<code>mandatory</code></br>
<em>
bool
</em>
</td>
<td>
<p>Mandatory marks this secret as being mandatory</p>
</td>
</tr>
<tr>
<td>
<code>gcpSecretsManager</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.GcpSecretsManager">
GcpSecretsManager
</a>
</em>
</td>
<td>
<p>GcpSecretsManager config</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.SourceConfigSpec">SourceConfigSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SourceConfig">SourceConfig</a>)
</p>
<p>
<p>SourceConfigSpec defines the desired state of SourceConfig.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>groups</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.RepositoryGroup">
[]RepositoryGroup
</a>
</em>
</td>
<td>
<p>Groups the groups of source repositories</p>
</td>
</tr>
<tr>
<td>
<code>scheduler</code></br>
<em>
string
</em>
</td>
<td>
<p>Scheduler the default scheduler for any group/repository which does not specify one</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsServers</code></br>
<em>
<a href="#gitops.jenkins-x.io/v1alpha1.JenkinsServer">
[]JenkinsServer
</a>
</em>
</td>
<td>
<p>JenkinsServers the jenkins servers configured for this repository</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsFolderTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsFolderTemplate the default template file to use to generate the folder job DSL script</p>
</td>
</tr>
<tr>
<td>
<code>jenkinsJobTemplate</code></br>
<em>
string
</em>
</td>
<td>
<p>JenkinsJobTemplate the default template file to use to generate the projects job DSL script</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>0efd3ac</code>.
</em></p>
