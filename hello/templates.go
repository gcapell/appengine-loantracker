package hello

import (
	"html/template"
)

var paymentTemplate = template.Must(template.New("book").Parse(paymentTemplateHTML))

const paymentTemplateHTML = `
<html>
	<body>
	<table border="1">
	{{range .}}
		<tr>
			<td>{{.Date.Format "2 Jan 2006"}}</td>
			<td align="right">
			{{if .Type.IsPayment}}
				Payment: {{.Amount}}
			{{end}}
			{{if .Type.IsInterest}}
				Interest: {{.Amount}}
			{{end}}
			{{if .Type.IsLoan}}
				Loan: {{.Amount}}
			{{end}}
			{{if .Type.IsRateChange}}
				Rate change to: {{.Rate}}%
			{{end}}
			</td>
			<td>{{.Owed}}</td>
		</tr>
	{{end}}
	</table>
    <form action="/addPayment" method="post">
	<table>
	 <tr><td>Date: </td><td><input type="text" name="date"></td></tr>
	 <tr>
		<td> Amount: </td><td><input type="text" name="amount"></td>
		<td><input type="checkbox" name="IsLoan">IsLoan</td>
	</tr>
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
 <tr><td> Interest Rate: </td><td><input type="text" name="rate"></td>

</tr>
 <tr><td>Date: </td><td><input type="text" name="date"></td></tr>
</table>
  <div><input type="submit" value="Update interest rate"></div>
</form>
</body>
</html>
`