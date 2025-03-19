
# fscopy

Parts:

- select files
- copy files
- link files

```mermaid
graph LR
    A[selectFiles] -->|list of files| B[copyFiles]
    B -->|link targets| C[linkFiles]
```
