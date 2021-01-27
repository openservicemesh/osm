# Open Service Mesh Docs

> :book: OSM Docs are found within this [section](https://github.com/openservicemesh/osm/tree/main/docs/content)
> :ship: Also the config to generate them as docs.openservicemesh.io
 
> :link: Looking for the main OSM website? Visit [osm-www](https://github.com/openservicemesh/osm-www).


## Editing Content

docs.openservicemesh.io is a static site. The documentation content needs to be located at `docs/content/docs/`.

To ensure the docs content renders correctly in the theme, each page will need to have [front matter](https://gohugo.io/content-management/front-matter/) metadata. Example front matter:

```
---
title: "Docs Home"
linkTitle: "Home"
description: "OSM Docs Home"
weight: 1
type: docs
---
```

## Front Matter Notes:

* inclusion of `type: docs` is important for the theme to properly index the site contents
* the `linkTitle` attribute allows you to simplify the name as it appears in the left-side nav bar - ideally it should be short and clear - whereas the title can handle longform names for pages/documents.

## Versioning the Docs Site

When a new OSM release is cut, the docs site should be updated to reflect the version change. The underlying Docsy theme has versioning support built-in.

<img width="244" alt="Screen Shot 2020-10-27 at 11 25 23 AM" src="https://user-images.githubusercontent.com/686194/97345732-a979ab80-1847-11eb-8c42-1b52c422a722.png">

## Release and Versioning Process:

1. Create an archive of the current `main` branch, following the naming convention used by [prior releases](https://github.com/openservicemesh/osm/branches). Push this archival branch to `upstream`.

<img width="301" alt="Screen Shot 2020-10-27 at 11 11 17 AM" src="https://user-images.githubusercontent.com/686194/97343954-5999e500-1845-11eb-96f4-d9d59352a830.png">

2. Once a release version branch is pushed to `upstream`, Netlify will build and deploy the documentation found within that branch. Within a couple of minutes of the branch push, the version of the site should be accessible at `https://<BRANCHNAME>--osm-docs.netlify.app/`

Test the url for the branch once deployed to ensure it is working.

3. Update the site version menu to reflect the changes

<img width="723" alt="Screen Shot 2020-10-27 at 11 36 02 AM" src="https://user-images.githubusercontent.com/686194/97346387-9ca98780-1848-11eb-8179-523dcbed79c0.png">

The version dropdown menu in set in the sites' `config.toml` (L73) file - adding a new `params.versions` archival entry for the prior release/branch, using the url from Netlify. The current release at the top of the list should reflect (i) the new version and (ii) the primary docs website url.

Example _(where v5 is new and v4 is now an archival prior release):_

```
[[params.versions]]
  version = "v0.5"
  url = "https://docs.openservicemesh.io"

[[params.versions]]
  version = "v0.4"
  url = "https://release-v0.4--osm-docs.netlify.app"
```

See [the Doscy versioning docs](https://www.docsy.dev/docs/adding-content/versioning/) for more information on these theme config parameters.

> Note: the versioning controls are forward-facing only due to the nature of git branch history. The documentation website and netlify build process was added after `release-v0.4`, so only future releases will be deployed in this manner. The first release to be properly archived in this way will be `release-v0.5`. Trying to access archival urls for earlier versions (0.1 to 0.4) will not work because they predate this documentation website.



# Site Development

## Notes

* built with the [Hugo](https://gohugo.io/) static site generator
* custom theme uses [Docsy](https://www.docsy.dev/) as a base, with [Bootstrap](https://getbootstrap.com/) as the underlying css framework and some [OSM custom sass](https://github.com/openservicemesh/osm/pull/1840/files#diff-374e78d353e95d66afe7e6c3e13de4aab370ffb117f32aeac519b15c2cbd057aR1)
* deployed to [Netlify](https://app.netlify.com/sites/osm-docs/deploys) via merges to main. (@flynnduism can grant additional access to account)
* metrics tracked via Google Analytics

## Install dependencies:

* Hugo [installation guide](https://gohugo.io/getting-started/installing/)
* packages are installed by running `yarn`

## Run the site:

```
// rebuild the site (to compile latest css/js)
hugo

// or serve the site for local dev
hugo serve
```

## Deploying the site:

The site auto deploys the main branch via [Netlify](https://app.netlify.com/sites/osm-docs). Once pull requests are merged the changes will appear at docs.openservicemesh.io after a couple of minutes. Check the [logs](https://app.netlify.com/sites/osm-docs/deploys) for details.

[![Netlify Status](https://api.netlify.com/api/v1/badges/8c8b7b52-b87f-42e0-949a-a784c3ca6d9a/deploy-status)](https://app.netlify.com/sites/osm-docs/deploys)

`hugo serve` will run the site locally at [localhost:1313](http://localhost:1313/)
