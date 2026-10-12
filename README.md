# File Organizer

Organizador de arquivos por regras para Omarchy. O núcleo é escrito em Go; a janela usa Qt Quick. Um widget da barra do Omarchy abre o aplicativo.

## O que já funciona

- Regras por extensão, nome exato ou parcial, padrão glob, regex, nome aproximado, tipo MIME e tamanho em bytes. A primeira regra correspondente vence.
- Prévia dos arquivos que serão movidos e dos conflitos de nome.
- Movimentação manual após confirmação, sem sobrescrever arquivos existentes.
- Regras e pastas salvas para a próxima abertura.
- Arquivos ocultos, links simbólicos e subpastas da origem são ignorados. A busca não é recursiva.

O aplicativo copia o arquivo para o destino e remove a origem após concluir a cópia. Isso permite mover entre sistemas de arquivos diferentes. Se a remoção da origem falhar, a cópia permanece no destino e o erro aparece no resultado.

## Como escrever regras

Uma regra precisa ter ao menos um critério. Campos preenchidos na mesma regra são combinados com **E**. Valores separados por vírgula em extensões ou tipos MIME são combinados com **OU**. Nome exato, glob e regex consideram letras maiúsculas e minúsculas; nome parcial e aproximado, extensões e tipos MIME não.

| Critério | Exemplo | Efeito |
| --- | --- | --- |
| Extensões | `pdf, docx` | Nome termina em uma dessas extensões. |
| Nome exato | `Relatório.pdf` | Nome completo é igual. |
| Nome contém | `fatura` | Corresponde a `Fatura-março.pdf`, sem distinguir maiúsculas. |
| Glob | `fatura-*.pdf` | `*`, `?` e classes como `[0-9]`. |
| Regex | `^IMG_[0-9]+\.jpg$` | Expressão regular de Go aplicada ao nome completo. |
| Nome aproximado | `relatorio.pdf`, até 1 diferença | Corresponde a `relatrio.pdf`. Compara o nome completo; 1 diferença é o padrão, com limite configurável de 1 a 3. |
| Tipos MIME | `image/*, application/pdf` | Tipo detectado pelo conteúdo. |
| Tamanho | mínimo `1024`, máximo `1048576` | Arquivo tem de 1 KiB a 1 MiB, inclusive. |

O tipo MIME é detectado a partir dos primeiros 512 bytes, sem confiar na extensão. Alguns formatos contêiner, como DOCX, podem ser identificados como ZIP. Para eles, use extensão ou nome quando precisar diferenciá-los.

## Desenvolvimento

Requisitos: Go 1.23+, CMake, Ninja e Qt 6.5+ com os módulos Quick e Quick Controls 2.

```sh
make build
make test
make run
```

O executável Qt e o núcleo Go ficam em `build/bin/`. A interface espera encontrar `fileorganizer-core` no mesmo diretório de `fileorganizer`.

`make test` executa os testes Go de regras, proteção dos arquivos e protocolo JSON. Também abre a janela Qt Quick em modo sem tela e verifica a comunicação com o núcleo nas ações de prévia e movimentação.

## Usar no Omarchy

Instale os dois executáveis no diretório pessoal:

```sh
make install
```

Depois que este repositório estiver publicado com o `manifest.json` e o `BarWidget.qml`, adicione o plugin:

```sh
omarchy plugin add https://github.com/KitsuneForgering/FileOrganizer.git --enable
```

O widget **Files** abre a janela pelo comando `fileorganizer`. O instalador de plugins do Omarchy não executa etapas de build; por isso os executáveis precisam estar instalados antes de usar o widget.

Para validar o manifesto localmente:

```sh
omarchy plugin validate .
```

## Protocolo do núcleo

O núcleo também pode ser chamado sem a interface. Ele lê um objeto JSON da entrada padrão e devolve o resultado em JSON:

```sh
printf '%s\n' '{"source":"/home/user/Downloads","destination":"/home/user/Organizados","rules":[{"extensions":["pdf","docx"],"folder":"Documentos"}]}' | build/bin/fileorganizer-core preview
```

Troque `preview` por `apply` para executar as regras. `apply` examina a pasta novamente; a lista pode mudar se arquivos forem adicionados ou removidos desde a prévia.
