export interface Adopter {
  name: string;
  url: string;
  img: string;
  description: string;
}

export const adopters: Adopter[] = [
  // Big brand names — first three double as the "featured" showcase
  {
    name: 'Docker',
    url: 'https://github.com/docker/mcp-registry',
    img: 'https://github.com/docker.png',
    description:
      'The industry-standard container platform uses Task in mcp-registry, the official registry for Docker Model Context Protocol servers.'
  },
  {
    name: 'Microsoft',
    url: 'https://github.com/Azure/Azure-Sentinel',
    img: 'https://github.com/microsoft.png',
    description:
      'Azure Sentinel — Microsoft’s cloud-native SIEM used by enterprises worldwide — relies on Task to orchestrate its repository automation.'
  },
  {
    name: 'HashiCorp',
    url: 'https://github.com/hashicorp/terraform-aws-terraform-enterprise-hvd',
    img: 'https://github.com/hashicorp.png',
    description:
      'HashiCorp ships Task across its Validated Design modules for Terraform, Vault, Consul, Nomad, and Boundary on AWS, Azure, and GCP.'
  },
  // Other big brands
  {
    name: 'Vercel',
    url: 'https://github.com/vercel/terraform-provider-vercel',
    img: 'https://github.com/vercel.png',
    description:
      'Vercel’s official Terraform provider uses Task as its development and release runner.'
  },
  {
    name: 'Google Cloud',
    url: 'https://github.com/GoogleCloudPlatform/deploystack',
    img: 'https://github.com/GoogleCloudPlatform.png',
    description:
      'DeployStack, Google Cloud’s one-click Terraform deployment tool, automates its workflows with Task.'
  },
  {
    name: 'AWS',
    url: 'https://github.com/aws-samples/appmod-blueprints',
    img: 'https://github.com/aws-samples.png',
    description:
      'The AWS Samples AppMod Blueprints reference platform uses Task to orchestrate its demo environments.'
  },
  {
    name: 'Anthropic',
    url: 'https://github.com/anthropics/buffa',
    img: 'https://github.com/anthropics.png',
    description:
      'Anthropic’s Rust protobuf implementation, buffa, uses Task for its build and release tooling.'
  },
  // Notable open source projects
  {
    name: 'Flet',
    url: 'https://github.com/flet-dev/flet',
    img: 'https://github.com/flet-dev.png',
    description:
      'Build realtime web, mobile and desktop apps in Python — with no frontend experience required.'
  },
  {
    name: 'GoReleaser',
    url: 'https://github.com/goreleaser/goreleaser',
    img: 'https://github.com/goreleaser.png',
    description:
      'Release engineering, simplified. GoReleaser is the de-facto release automation tool for Go projects.'
  },
  {
    name: 'Arduino CLI',
    url: 'https://github.com/arduino/arduino-cli',
    img: 'https://github.com/arduino.png',
    description:
      'The official Arduino command-line tool. Task powers the entire Arduino developer tooling stack across 70+ repositories.'
  },
  {
    name: 'FerretDB',
    url: 'https://github.com/FerretDB/FerretDB',
    img: 'https://github.com/FerretDB.png',
    description:
      'A truly open-source MongoDB alternative built on PostgreSQL, with Task driving every build and release step.'
  },
  {
    name: 'Tyk',
    url: 'https://github.com/TykTechnologies/tyk',
    img: 'https://github.com/TykTechnologies.png',
    description:
      'Open source API gateway supporting REST, GraphQL, TCP and gRPC — automated end-to-end with Task.'
  },
  {
    name: 'Charmbracelet',
    url: 'https://github.com/charmbracelet/glamour',
    img: 'https://github.com/charmbracelet.png',
    description:
      'The team behind Bubble Tea uses Task to build Glamour, the stylesheet-based markdown renderer for CLI apps.'
  },
  {
    name: 'Outline',
    url: 'https://github.com/OutlineFoundation/outline-server',
    img: 'https://github.com/OutlineFoundation.png',
    description:
      'Outline — the open-source proxy server originally built by Jigsaw (Google) — uses Task for its build pipeline.'
  }
];
