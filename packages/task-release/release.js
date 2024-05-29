import { spawn as spawnCb } from "node:child_process";
import { readFileSync } from "node:fs";
import { cp, mkdir, readdir, rm, writeFile } from "node:fs/promises";
import mainPackage from "../../package.json" assert { type: "json" };

function spawn(command, args, options) {
  return new Promise((resolve, reject) => {
    const child = spawnCb(command, args, options);
    child.on("exit", (code) => {
      if (code === 0) {
        return resolve();
      }
      reject(new Error(`Process exited with code ${code}`));
    })
  });
}

export const platforms = [
  {
    os: "darwin",
    arch: "arm64",
    exe: "./bin/task",
  },
  {
    os: "darwin",
    goarch: "amd64",
    arch: "x64",
    exe: "./bin/task",
  },
  {
    os: "linux",
    arch: "arm64",
    exe: "./bin/task",
  },
  {
    os: "linux",
    goarch: "amd64",
    arch: "x64",
    exe: "./bin/task",
  },
  {
    os: "win32",
    goos: "windows",
    goarch: "amd64",
    arch: "x64",
    exe: "./bin/task.exe",
  },
];

const packageJson = readFileSync("package.json", "utf-8");

async function createFolder(platform) {
  const folder = `task-${platform.os}-${platform.arch}`

  console.debug(`Creating folder ${folder}`)
  await mkdir(`${folder}/bin`, { recursive: true })
  console.debug(`Copying files to ${folder}`)
  await writeFile(`${folder}/package.json`, packageJson.replace(/{platform}/g, platform.os).replace(/{arch}/g, platform.arch).replace(/{version}/g, mainPackage.version))
  await cp('../../README.md', `${folder}/README.md`)
  await cp('../../LICENSE', `${folder}/LICENSE`)
  console.debug(`Building ${folder}`)
  await spawn("go", ["build", "-o", `${folder}/${platform.exe}`, "-v", "../../cmd/task"], {
    env: {
      ...process.env,
      GOOS: platform.goos || platform.os,
      GOARCH: platform.goarch || platform.arch,
    },
  })
  return folder;
}

async function publish(platform) {
  const folder = await createFolder(platform)
  console.debug(`Publishing ${folder}`)
  await spawn("npm", ["publish", '--provenance', '--access', 'public', process.env.DRY_RUN ? '--dry-run' : ''], { stdio: "inherit", cwd: folder })
}

async function main() {
  const list = await readdir('.', { withFileTypes: true });
  await Promise.all(list.filter((entry) => entry.isDirectory() && entry.name.startsWith('task-')).map((entry) => rm(entry.name, { recursive: true, force: true })));
  for (const platform of platforms) {
    await publish(platform)
  }
}

main();
