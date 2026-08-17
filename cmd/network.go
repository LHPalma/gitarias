package cmd

import "time"

// networkDeadline existe porque toda outra operação do gtr é local e termina
// sozinha. Uma chamada de rede não termina: sem prazo, um servidor calado
// deixaria o comando pendurado para sempre.
const networkDeadline = 30 * time.Second
