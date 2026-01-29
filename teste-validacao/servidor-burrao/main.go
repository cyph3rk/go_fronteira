package main

import (
	"fmt"
	"net/http"
)

func main() {	
	http.HandleFunc("/showTela", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>Tela do Sistema</h1><p>Requisição recebida com sucesso!</p>")		
		fmt.Println("Log: Alguém acessou o endpoint /showTela")
	})
	fmt.Println("Servidor rodando em http://localhost:8081")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		fmt.Printf("Erro ao subir o servidor: %s\n", err)
	}
}