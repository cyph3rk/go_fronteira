// utilitário pequeno para formatação rápida/consistente de valores numéricos em headers/logs.
//    Evita puxar fmt (que é mais “pesado” e genérico) só para formatação simples
// 	  Padroniza a formatação do float (strconv.FormatFloat), evitando notação científica em 
//        valores comuns e mantendo o código consistente

package ratelimit

import "strconv"

func formatInt(v int) string { return strconv.Itoa(v) }

func formatFloat(v float64) string {
	// sem depender de fmt, e sem notação científica para valores comuns
	return strconv.FormatFloat(v, 'f', -1, 64)
}
