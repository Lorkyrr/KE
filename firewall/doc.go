// Package firewall é a única entrada e a única saída de dados do KE (spec
// v2 §3, D1 e D2). É um módulo: sanitizar (folhas puras), entrada (contrato
// declarado por tag) e router (externo transporta, interno encaminha).
//
// O dado não é copiado entre as etapas do firewall: o router lê do
// transporte uma vez para dentro de um quadro, a entrada valida pelas faixas
// e o interno passa adiante só o endereço do quadro.
package firewall
