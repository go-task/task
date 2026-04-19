export interface Adopter {
  name: string;
  url: string;
  img: string;
}

export const adopters: Adopter[] = [
  // Big brand names
  {
    name: 'Docker',
    url: 'https://github.com/docker/mcp-registry',
    img: 'https://github.com/docker.png'
  },
  {
    name: 'HashiCorp',
    url: 'https://github.com/hashicorp/terraform-aws-terraform-enterprise-hvd',
    img: 'https://github.com/hashicorp.png'
  },
  {
    name: 'Microsoft',
    url: 'https://github.com/microsoft/terraform-provider-fabric',
    img: 'https://github.com/microsoft.png'
  },
  {
    name: 'Vercel',
    url: 'https://github.com/vercel/terraform-provider-vercel',
    img: 'https://github.com/vercel.png'
  },
  {
    name: 'Google Cloud',
    url: 'https://github.com/GoogleCloudPlatform/deploystack',
    img: 'https://github.com/GoogleCloudPlatform.png'
  },
  {
    name: 'AWS',
    url: 'https://github.com/aws-samples/appmod-blueprints',
    img: 'https://github.com/aws-samples.png'
  },
  // Notable open source projects
  {
    name: 'Arduino CLI',
    url: 'https://github.com/arduino/arduino-cli',
    img: 'https://github.com/arduino.png'
  },
  {
    name: 'GoReleaser',
    url: 'https://github.com/goreleaser/goreleaser',
    img: 'https://github.com/goreleaser.png'
  },
  {
    name: 'FerretDB',
    url: 'https://github.com/FerretDB/FerretDB',
    img: 'https://github.com/FerretDB.png'
  },
  {
    name: 'Gogs',
    url: 'https://github.com/gogs/gogs',
    img: 'https://github.com/gogs.png'
  },
  {
    name: 'Tyk',
    url: 'https://github.com/TykTechnologies/tyk',
    img: 'https://github.com/TykTechnologies.png'
  },
  {
    name: 'Outline',
    url: 'https://github.com/OutlineFoundation/outline-server',
    img: 'https://github.com/OutlineFoundation.png'
  }
];
