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
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMapping">SecretMapping</a>
</li></ul>
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
<code>defaultBackendType</code></br>
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
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="gitops.jenkins-x.io/v1alpha1.BackendType">BackendType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#gitops.jenkins-x.io/v1alpha1.SecretMappingSpec">SecretMappingSpec</a>, 
<a href="#gitops.jenkins-x.io/v1alpha1.SecretRule">SecretRule</a>)
</p>
<p>
<p>BackendType describes a secrets backend</p>
</p>
<h3 id="gitops.jenkins-x.io/v1alpha1.GcpSecretsManager">GcpSecretsManager
</h3>
<p>
(<em>Appears on:</em>
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
<p>ProjectId for the secret</p>
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
<code>defaultBackendType</code></br>
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
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>3e27e20</code>.
</em></p>
