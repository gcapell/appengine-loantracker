package hello

import (
	"html/template"
)

var paymentTemplate = template.Must(template.New("book").Parse(paymentTemplateHTML))

const paymentTemplateHTML = `
<html>
  <body>
   <table>
    {{range .}}
	<tr>
	<td align="right">{{.Amount}}</td>
	<td>{{.Date.Format "15:04 2 Jan 2006"}}</td>
	</tr>
	{{end}}
	</table>
    <form action="/addPayment" method="post">
	<table>
	 <tr><td> Amount: </td><td><input type="text" name="amount"></tßd></tr>
	 <tr><td>Date: </td><td><input type="text" name="date"></td></tr>
	</table>
      <div><input type="submit" value="Add amount"></div>
    </form>
	<a href="/rate">Change interest rate</a>
  </body>
</html>
`
var rateTemplate = template.Must(template.New("rate").Parse(rateTemplateHTML))

const rateTemplateHTML = `
<html>
<body>
<form action="/changeRate" method="post">
<table>
 <tr><td> Interest Rate: </td><td><input type="text" name="rate"></td></tr>
 <tr><td>Date: </td><td><input type="text" name="date"></td></tr>
</table>
  <div><input type="submit" value="Update interest rate"></div>
</form>
</body>
</html>
`