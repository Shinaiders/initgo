#!/usr/bin/env node

/**
 * create-goinitshinaider / goinitshinaider
 * CLI para inicializar projetos Go a partir do template initgo (Shinaiders/initgo)
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const readline = require('readline');

const REPO_URL = 'https://github.com/Shinaiders/initgo.git';
const OLD_MODULE = 'goscrapper';

// Assegurar PATH completo para localizar go, git, etc. em macOS/Linux
const defaultEnv = {
  ...process.env,
  PATH: `/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:${process.env.PATH || ''}`,
};

// Cores para o terminal
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  cyan: '\x1b[36m',
  yellow: '\x1b[33m',
  red: '\x1b[31m',
  magenta: '\x1b[35m',
};

function log(msg, color = colors.reset) {
  console.log(`${color}${msg}${colors.reset}`);
}

function ask(question) {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

function runCommand(cmd, options = {}) {
  try {
    return execSync(cmd, { env: defaultEnv, stdio: 'inherit', ...options });
  } catch (err) {
    throw new Error(`Falha ao executar comando: ${cmd}`);
  }
}

function replaceInFile(filePath, search, replacement) {
  if (!fs.existsSync(filePath)) return;
  const content = fs.readFileSync(filePath, 'utf8');
  const updated = content.split(search).join(replacement);
  fs.writeFileSync(filePath, updated, 'utf8');
}

function walkDir(dir, callback) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name !== '.git' && entry.name !== 'node_modules') {
        walkDir(fullPath, callback);
      }
    } else {
      callback(fullPath);
    }
  }
}

async function main() {
  log('\n======================================================', colors.cyan);
  log('   🚀 Bem-vindo ao Inicializador de Projetos Go      ', colors.bright + colors.green);
  log('              Template: initgo                        ', colors.magenta);
  log('======================================================\n', colors.cyan);

  let targetInput = process.argv[2];

  if (!targetInput) {
    targetInput = await ask(
      `${colors.yellow}👉 Digite o nome do novo projeto (ex: meu-crawler ou github.com/user/meuapp): ${colors.reset}`
    );
  }

  if (!targetInput) {
    log('❌ Erro: O nome do projeto não pode ser vazio.', colors.red);
    process.exit(1);
  }

  // Determinar diretório e módulo
  let targetDir;
  let newModuleName;
  let baseAppName;

  // Se o usuário digitou algo como github.com/usuario/meu-projeto
  if (targetInput.includes('/') && !targetInput.startsWith('.') && !targetInput.startsWith('/')) {
    newModuleName = targetInput;
    baseAppName = path.basename(targetInput);
    targetDir = path.resolve(process.cwd(), baseAppName);
  } else {
    targetDir = path.resolve(process.cwd(), targetInput);
    baseAppName = path.basename(targetDir);
    newModuleName = baseAppName;
  }

  if (fs.existsSync(targetDir)) {
    const files = fs.readdirSync(targetDir);
    if (files.length > 0) {
      log(`❌ Erro: O diretório '${targetDir}' já existe e não está vazio.`, colors.red);
      process.exit(1);
    }
  }

  log(`\n📥 1. Clonando template para ${colors.bright}${targetDir}${colors.reset}...`, colors.cyan);
  runCommand(`git clone --depth 1 ${REPO_URL} "${targetDir}"`, { stdio: 'pipe' });

  log('🧹 2. Desacoplando repositório git...', colors.cyan);
  fs.rmSync(path.join(targetDir, '.git'), { recursive: true, force: true });

  log(`🔧 3. Renomeando referências de '${OLD_MODULE}' para '${newModuleName}'...`, colors.cyan);

  // go.mod
  const goModPath = path.join(targetDir, 'go.mod');
  if (fs.existsSync(goModPath)) {
    replaceInFile(goModPath, `module ${OLD_MODULE}`, `module ${newModuleName}`);
  }

  // Arquivos Go (.go)
  walkDir(targetDir, (filePath) => {
    if (filePath.endsWith('.go')) {
      replaceInFile(filePath, `"${OLD_MODULE}/`, `"${newModuleName}/`);
    }
  });

  // Makefile
  const makefilePath = path.join(targetDir, 'Makefile');
  if (fs.existsSync(makefilePath)) {
    replaceInFile(makefilePath, `APP_NAME=${OLD_MODULE}`, `APP_NAME=${baseAppName}`);
  }

  // docker-compose.yml
  const composePath = path.join(targetDir, 'docker-compose.yml');
  if (fs.existsSync(composePath)) {
    replaceInFile(composePath, `${OLD_MODULE}_`, `${baseAppName}_`);
    replaceInFile(composePath, `${OLD_MODULE}-`, `${baseAppName}-`);
    replaceInFile(composePath, `${OLD_MODULE}.db`, `${baseAppName}.db`);
  }

  // .env e .env.example
  const envExamplePath = path.join(targetDir, '.env.example');
  const envPath = path.join(targetDir, '.env');
  if (fs.existsSync(envExamplePath)) {
    replaceInFile(envExamplePath, `${OLD_MODULE}_`, `${baseAppName}_`);
    replaceInFile(envExamplePath, `${OLD_MODULE}.db`, `${baseAppName}.db`);
    fs.copyFileSync(envExamplePath, envPath);
  }

  // Inicializar novo repositório Git
  log('🌱 4. Inicializando novo repositório Git...', colors.cyan);
  try {
    runCommand('git init', { cwd: targetDir, stdio: 'pipe' });
    runCommand('git add .', { cwd: targetDir, stdio: 'pipe' });
    runCommand('git commit -m "feat: initial commit from initgo template"', { cwd: targetDir, stdio: 'pipe' });
  } catch (err) {
    log('⚠️ Aviso: Não foi possível criar commit inicial do git automaticamente.', colors.yellow);
  }

  // Executar go mod tidy se Go estiver instalado
  log('📦 5. Verificando dependências do Go...', colors.cyan);
  try {
    runCommand('go version', { stdio: 'pipe' });
    runCommand('go mod tidy', { cwd: targetDir, stdio: 'pipe' });
    log('   ✓ Dependências Go sincronizadas com sucesso.', colors.green);
  } catch (err) {
    log('   ℹ️ Go não encontrado ou offline. Execute "go mod tidy" posteriormente.', colors.yellow);
  }

  log('\n======================================================', colors.green);
  log('   🎉 Projeto criado e configurado com sucesso!       ', colors.bright + colors.green);
  log('======================================================\n', colors.green);

  log('Para começar a desenvolver:', colors.bright);
  log(`  cd ${path.relative(process.cwd(), targetDir) || baseAppName}`, colors.cyan);
  log('  make deps-up   # (sobe PostgreSQL, Redis, RabbitMQ e RustFS)', colors.cyan);
  log('  make dev       # (inicia o servidor com live-reload)', colors.cyan);
  log('\nDivirta-se codando! 🚀\n', colors.bright + colors.magenta);
}

main().catch((err) => {
  log(`\n❌ Ocorreu um erro: ${err.message}`, colors.red);
  process.exit(1);
});
